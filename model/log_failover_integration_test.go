package model

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/url"
	"os"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	mysqldriver "github.com/go-sql-driver/mysql"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// A controllable TCP fault lets the production SQL drivers experience actual
// broken connections without stopping a developer's database server.
type logFaultProxy struct {
	listener    net.Listener
	target      string
	mu          sync.Mutex
	blocked     bool
	connections map[net.Conn]struct{}
	wg          sync.WaitGroup
}

func newLogFaultProxy(t *testing.T, target string, blocked bool) *logFaultProxy {
	t.Helper()
	l, err := net.Listen("tcp", "127.0.0.1:0")
	require.NoError(t, err)
	p := &logFaultProxy{listener: l, target: target, blocked: blocked, connections: map[net.Conn]struct{}{}}
	p.wg.Add(1)
	go func() {
		defer p.wg.Done()
		for {
			client, err := l.Accept()
			if err != nil {
				return
			}
			p.mu.Lock()
			if p.blocked {
				p.mu.Unlock()
				client.Close()
				continue
			}
			p.connections[client] = struct{}{}
			p.mu.Unlock()
			p.wg.Add(1)
			go func() {
				defer p.wg.Done()
				defer func() { client.Close(); p.mu.Lock(); delete(p.connections, client); p.mu.Unlock() }()
				server, err := net.DialTimeout("tcp", target, time.Second)
				if err != nil {
					return
				}
				defer server.Close()
				copyDone := make(chan struct{})
				go func() { io.Copy(server, client); server.Close(); close(copyDone) }()
				io.Copy(client, server)
				client.Close()
				<-copyDone
			}()
		}
	}()
	t.Cleanup(func() { l.Close(); p.SetBlocked(true); p.wg.Wait() })
	return p
}

func (p *logFaultProxy) SetBlocked(blocked bool) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.blocked = blocked
	if blocked {
		for conn := range p.connections {
			conn.Close()
		}
	}
}

func failoverSQLTestDB(t *testing.T) *gorm.DB {
	t.Helper()
	dsn := os.Getenv("TEST_LOG_PRIMARY_DSN")
	var dialect gorm.Dialector = sqlite.Open(":memory:")
	if strings.HasPrefix(dsn, "postgres") {
		u, err := url.Parse(dsn)
		require.NoError(t, err)
		require.Equal(t, "/new_api_test", u.Path)
		dialect = postgres.Open(dsn)
	} else if dsn != "" {
		cfg, err := mysqldriver.ParseDSN(dsn)
		require.NoError(t, err)
		require.Equal(t, "new_api_test", cfg.DBName)
		dialect = mysql.Open(dsn)
	}
	db, err := gorm.Open(dialect, &gorm.Config{})
	require.NoError(t, err)
	t.Cleanup(func() { closeDB(db) })
	require.NoError(t, db.AutoMigrate(&Log{}))
	require.NoError(t, db.Exec("DELETE FROM logs").Error)
	return db
}

func failoverTestStore(t *testing.T, startDown bool) (*clickHouseLogStore, *logFaultProxy, *clickHouseLogStore, *gorm.DB) {
	t.Helper()
	primaryDSN := os.Getenv("TEST_CLICKHOUSE_DSN")
	if primaryDSN == "" {
		t.Skip("set TEST_CLICKHOUSE_DSN for disaster tests; TEST_LOG_PRIMARY_DSN selects MySQL/PostgreSQL")
	}
	fallback := failoverSQLTestDB(t)
	u, err := url.Parse(primaryDSN)
	require.NoError(t, err)
	u.Path = fmt.Sprintf("/new_api_test_%d", time.Now().UnixNano())
	direct, err := newClickHouseLogStore(context.Background(), u.String(), 30)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := direct.db.Exec("DROP DATABASE " + quoteClickHouseIdentifier(direct.database) + " SYNC")
		require.NoError(t, err)
		direct.Close()
	})
	proxy := newLogFaultProxy(t, u.Host, startDown)
	u.Host = proxy.listener.Addr().String()
	oldDB, oldLogs, oldStore, oldCH := DB, LOG_DB, clickHouseLogs, common.UsingClickHouse
	DB, LOG_DB, clickHouseLogs = fallback, nil, nil
	t.Setenv("CLICKHOUSE_DSN", u.String())
	t.Setenv("CLICKHOUSE_NODE_ID", "31")
	t.Setenv("LOG_SQL_DSN", "")
	t.Cleanup(func() {
		if clickHouseLogs != nil {
			clickHouseLogs.Close()
		}
		DB, LOG_DB, clickHouseLogs, common.UsingClickHouse = oldDB, oldLogs, oldStore, oldCH
	})
	require.NoError(t, InitLogDB())
	require.Same(t, fallback, LOG_DB)
	require.Same(t, fallback, DB)
	return clickHouseLogs, proxy, direct, fallback
}

func TestClickHouseRuntimeOriginalDatabaseFailover(t *testing.T) {
	s, proxy, direct, fallback := failoverTestStore(t, false)
	ctx := context.Background()
	history := Log{Content: "existing history"}
	require.NoError(t, fallback.Create(&history).Error)
	log := func(content string) *Log {
		return &Log{UserId: 10, CreatedAt: time.Now().Unix(), Type: LogTypeConsume, Content: content, Quota: 123, Other: `{"preserve":true}`, RequestId: content}
	}
	require.NoError(t, createLog(ctx, log("healthy")))
	proxy.SetBlocked(true)
	outageOne, outageTwo := log("outage-one"), log("outage-two")
	wantOne, wantTwo := *outageOne, *outageTwo
	require.NoError(t, createLog(ctx, outageOne))
	require.NoError(t, createLog(ctx, outageTwo))
	var backup []Log
	require.NoError(t, fallback.Order("id").Find(&backup).Error)
	require.Len(t, backup, 3)
	wantOne.Id, wantTwo.Id = backup[1].Id, backup[2].Id
	require.Equal(t, []Log{history, wantOne, wantTwo}, backup)
	require.Greater(t, backup[1].Id, history.Id)
	readState := &LogReadState{}
	stored, originalTotal, err := s.Query(logReadContext(readState), logFilter{}, 0, 10, logOrderID)
	require.NoError(t, err)
	require.Equal(t, int64(3), originalTotal)
	require.Len(t, stored, 3)
	require.Equal(t, "original", readState.Source())
	proxy.SetBlocked(false)
	time.Sleep(logFailoverRetryInterval + 10*time.Millisecond)
	require.NoError(t, createLog(ctx, log("recovered")))
	rows, total, err := direct.Query(ctx, logFilter{}, 0, 10, logOrderID)
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, rows, 2)
	readState = &LogReadState{}
	_, recoveredTotal, err := s.Query(logReadContext(readState), logFilter{}, 0, 10, logOrderID)
	require.NoError(t, err)
	require.Equal(t, int64(2), recoveredTotal)
	require.Equal(t, "clickhouse", readState.Source())
	var backupCount int64
	require.NoError(t, fallback.Model(&Log{}).Count(&backupCount).Error)
	require.Equal(t, int64(3), backupCount)
	proxy.SetBlocked(true)
	require.NoError(t, closeDB(fallback))
	err = s.Create(ctx, log("both-down"))
	require.ErrorContains(t, err, "original log database write failed")
}

func TestClickHouseStartupCanUseOriginalDatabase(t *testing.T) {
	s, proxy, direct, fallback := failoverTestStore(t, true)
	log := &Log{UserId: 7, CreatedAt: time.Now().Unix(), Type: LogTypeError, Content: "startup outage", Other: "raw"}
	require.NoError(t, s.Create(context.Background(), log))
	var saved Log
	require.NoError(t, fallback.First(&saved).Error)
	require.Equal(t, log.Content, saved.Content)
	proxy.SetBlocked(false)
	time.Sleep(logFailoverRetryInterval + 10*time.Millisecond)
	require.NoError(t, s.Create(context.Background(), &Log{CreatedAt: time.Now().Unix(), Content: "startup recovered"}))
	_, total, err := direct.Query(context.Background(), logFilter{}, 0, 10, logOrderID)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
}

func TestClickHouseOutagePersistsEveryRequestLog(t *testing.T) {
	s, proxy, direct, fallback := failoverTestStore(t, false)
	proxy.SetBlocked(true)
	const count = 100
	want := make([]Log, count)
	errCh := make(chan error, count)
	var wg sync.WaitGroup
	for i := range want {
		want[i] = Log{
			UserId: 7, CreatedAt: time.Now().Unix(), Type: LogTypeConsume,
			Content: "灾备落库校验", Username: "test-user", TokenName: "test-token",
			ModelName: "test-model", Quota: 128, PromptTokens: 31, CompletionTokens: 17,
			UseTime: 2, IsStream: true, ChannelId: 3, TokenId: 4, Group: "default",
			Ip: "127.0.0.1", RequestId: fmt.Sprintf("outage-%03d", i),
			UpstreamRequestId: fmt.Sprintf("upstream-%03d", i), Other: `{"token_audit":{"masked_key":"sk-test"},"model_ratio":1,"unknown":{"preserve":true}}`,
		}
		wg.Add(1)
		go func(log Log) {
			defer wg.Done()
			errCh <- createLog(context.Background(), &log)
		}(want[i])
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		require.NoError(t, err)
	}
	var saved []Log
	require.NoError(t, fallback.Order("request_id").Find(&saved).Error)
	require.Len(t, saved, count)
	for i := range want {
		require.Positive(t, saved[i].Id)
		want[i].Id = saved[i].Id
	}
	require.Equal(t, want, saved, "every request and all business fields must survive the outage")
	proxy.SetBlocked(false)
	s.failover.mu.Lock()
	s.failover.retryAt = time.Now().Add(-time.Second)
	s.failover.mu.Unlock()
	require.NoError(t, createLog(context.Background(), &Log{CreatedAt: time.Now().Unix(), RequestId: "after-recovery"}))
	rows, total, err := direct.Query(context.Background(), logFilter{}, 0, 10, logOrderID)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Equal(t, "after-recovery", rows[0].RequestId)
	var remaining int64
	require.NoError(t, fallback.Model(&Log{}).Count(&remaining).Error)
	require.Equal(t, int64(count), remaining, "recovery must retain SQL records for later migration")
	t.Logf("verified %d/%d complete request logs in %s; recovery kept all SQL records", count, count, fallback.Dialector.Name())
}

func TestClickHouseCloseDoesNotCloseOriginalDatabase(t *testing.T) {
	s, _, _, fallback := failoverTestStore(t, false)
	require.NoError(t, s.Close())
	require.NoError(t, fallback.Create(&Log{Content: "original connection still works"}).Error)
}

func TestClickHouseStartupWithUnavailableOriginalDatabase(t *testing.T) {
	_, proxy, _, fallback := failoverTestStore(t, false)
	dsn := os.Getenv("CLICKHOUSE_DSN")
	require.NoError(t, closeDB(fallback))
	active, err := newClickHouseLogStoreWithFallback(context.Background(), dsn, 32, fallback)
	require.NoError(t, err)
	defer active.Close()
	require.NoError(t, active.Create(context.Background(), &Log{Content: "primary remains usable", CreatedAt: time.Now().Unix()}))
	proxy.SetBlocked(true)
	_, err = newClickHouseLogStoreWithFallback(context.Background(), dsn, 33, fallback)
	require.ErrorContains(t, err, "original log database unavailable")
	require.ErrorContains(t, err, "ClickHouse logs unavailable")
}

func TestInitLogDBClickHouseFallsBackToExistingLogSQLDSN(t *testing.T) {
	logDSN := os.Getenv("TEST_LOG_PRIMARY_DSN")
	if logDSN == "" {
		t.Skip("set TEST_LOG_PRIMARY_DSN for the existing separate logs connection test")
	}
	_, dsn := clickHouseTestDatabase(t)
	originalLogs := failoverSQLTestDB(t)
	mainDB, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	t.Cleanup(func() { closeDB(mainDB) })
	require.NoError(t, mainDB.AutoMigrate(&Log{}))
	history := Log{Content: "main database unchanged"}
	require.NoError(t, mainDB.Create(&history).Error)
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	proxy := newLogFaultProxy(t, u.Host, false)
	u.Host = proxy.listener.Addr().String()
	oldDB, oldLogs, oldStore, oldCH, oldLogType := DB, LOG_DB, clickHouseLogs, common.UsingClickHouse, common.LogSqlType
	DB, LOG_DB, clickHouseLogs = mainDB, nil, nil
	t.Setenv("CLICKHOUSE_DSN", u.String())
	t.Setenv("CLICKHOUSE_NODE_ID", "34")
	t.Setenv("LOG_SQL_DSN", logDSN)
	t.Cleanup(func() {
		if clickHouseLogs != nil {
			clickHouseLogs.Close()
		}
		if LOG_DB != nil && LOG_DB != mainDB {
			closeDB(LOG_DB)
		}
		DB, LOG_DB, clickHouseLogs, common.UsingClickHouse, common.LogSqlType = oldDB, oldLogs, oldStore, oldCH, oldLogType
		initCol()
	})
	require.NoError(t, InitLogDB())
	require.Same(t, mainDB, DB)
	require.NotSame(t, mainDB, LOG_DB)
	proxy.SetBlocked(true)
	log := Log{CreatedAt: time.Now().Unix(), Content: "goes to original log SQL DSN", Group: "default", Other: "raw"}
	require.NoError(t, createLog(context.Background(), &log))
	var stored []Log
	require.NoError(t, originalLogs.Find(&stored).Error)
	require.Len(t, stored, 1)
	log.Id = stored[0].Id
	require.Equal(t, []Log{log}, stored)
	require.NoError(t, mainDB.Find(&stored).Error)
	require.Equal(t, []Log{history}, stored)
}
