package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

// Set once during InitLogDB, before serving requests. A nil store preserves the
// existing LOG_DB implementation and its independently configured SQL dialect.
var clickHouseLogs *clickHouseLogStore

type clickHouseLogStore struct {
	db          *sql.DB
	database    string
	ids         *logIDGenerator
	node        int
	initMu      sync.Mutex
	failover    *logWriteFailover
	engine      string
	fallback    *gorm.DB
	readRetryAt atomic.Int64
}

var clickHouseIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func quoteClickHouseIdentifier(s string) string { return "`" + s + "`" }

func (s *clickHouseLogStore) table() string {
	return quoteClickHouseIdentifier(s.database) + ".`logs`"
}

func newClickHouseLogStore(ctx context.Context, dsn string, node int) (*clickHouseLogStore, error) {
	s, err := openClickHouseLogStore(dsn, node)
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if err := s.prepare(ctx); err != nil {
		s.db.Close()
		return nil, err
	}
	return s, nil
}

func openClickHouseLogStore(dsn string, node int) (*clickHouseLogStore, error) {
	// Validate without echoing the DSN: url.Parse errors can contain passwords.
	u, err := url.Parse(strings.TrimSpace(dsn))
	if err != nil || u == nil || (u.Scheme != "clickhouse" && u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, errors.New("invalid CLICKHOUSE_DSN (expected clickhouse://, http:// or https://)")
	}
	for _, address := range strings.Split(u.Host, ",") {
		_, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, errors.New("CLICKHOUSE_DSN must specify a valid host and port")
		}
		if n, err := strconv.Atoi(port); err != nil || n < 1 || n > 65535 {
			return nil, errors.New("invalid CLICKHOUSE_DSN port")
		}
	}
	options, err := clickhouse.ParseDSN(strings.TrimSpace(dsn))
	if err != nil {
		return nil, errors.New("invalid CLICKHOUSE_DSN options")
	}
	database := options.Auth.Database
	if u.Path == "" || u.Path == "/" {
		database = "logs"
	}
	if !clickHouseIdentifier.MatchString(database) {
		return nil, errors.New("ClickHouse database names must contain only letters, digits and underscores")
	}
	if node < 0 || node > 1023 {
		return nil, errors.New("CLICKHOUSE_NODE_ID must be between 0 and 1023")
	}
	if strconv.IntSize != 64 {
		return nil, errors.New("ClickHouse logs require a 64-bit New API build")
	}
	if options.Settings == nil {
		options.Settings = clickhouse.Settings{}
	}
	// Wait for server-side insert batching to finish before reporting success.
	if _, configured := options.Settings["async_insert"]; !configured {
		options.Settings["async_insert"] = 1
		options.Settings["async_insert_busy_timeout_ms"] = 100
	}
	options.Settings["wait_for_async_insert"] = 1
	// A cluster entry must not acknowledge an insert before forwarding to its shard.
	options.Settings["insert_distributed_sync"] = 1
	options.Settings["skip_unavailable_shards"] = 0
	options.Debug = false
	if options.DialTimeout == 0 {
		options.DialTimeout = 10 * time.Second
	}
	// Connect to an existing database first, so the logs database can be created.
	options.Auth.Database = "default"
	db := clickhouse.OpenDB(options)
	return &clickHouseLogStore{db: db, database: database, node: node}, nil
}

func (s *clickHouseLogStore) prepare(ctx context.Context) error {
	s.initMu.Lock()
	defer s.initMu.Unlock()
	if s.ids != nil {
		return nil
	}
	if err := s.db.PingContext(ctx); err != nil {
		return fmt.Errorf("connect ClickHouse logs: %w", err)
	}
	if err := s.initialize(ctx); err != nil {
		return err
	}
	var maxID int64
	if err := s.db.QueryRowContext(ctx, "SELECT coalesce(max(id), 0) FROM "+s.table()).Scan(&maxID); err != nil {
		return fmt.Errorf("read ClickHouse log ID maximum: %w", err)
	}
	ids, err := newLogIDGenerator(s.node, maxID, time.Now)
	if err != nil {
		return err
	}
	s.ids = ids
	return nil
}

func newClickHouseLogStoreWithFallback(ctx context.Context, dsn string, node int, fallback *gorm.DB) (*clickHouseLogStore, error) {
	if fallback == nil {
		return nil, errors.New("original log database must be initialized before ClickHouse")
	}
	s, err := openClickHouseLogStore(dsn, node)
	if err != nil {
		return nil, err
	}
	// Reuse LOG_DB and its existing logs table; this store does not own the pool.
	s.fallback = fallback
	fallbackCtx, cancel := context.WithTimeout(ctx, logFailoverWriteTimeout)
	var existing []Log
	fallbackErr := fallback.WithContext(fallbackCtx).Select("id").Limit(1).Find(&existing).Error
	cancel()
	primaryCtx, cancel := context.WithTimeout(ctx, 10*time.Second)
	primaryErr := s.prepare(primaryCtx)
	cancel()
	if primaryErr != nil && fallbackErr != nil {
		s.Close()
		return nil, errors.Join(fmt.Errorf("ClickHouse logs unavailable: %w", primaryErr), fmt.Errorf("original log database unavailable: %w", fallbackErr))
	}
	s.failover = newLogWriteFailover(s.createDirect, func(ctx context.Context, log *Log) error {
		return saveLogToOriginalDatabase(ctx, fallback, log)
	})
	if primaryErr != nil {
		s.failover.unavailable(primaryErr)
	}
	if fallbackErr != nil {
		common.SysError("Original log database is not ready for failover; primary ClickHouse remains active: " + fallbackErr.Error())
	}
	return s, nil
}

func initClickHouseLogs(dsn string) error {
	node := 0
	if raw := os.Getenv("CLICKHOUSE_NODE_ID"); raw != "" {
		var err error
		node, err = strconv.Atoi(raw)
		if err != nil {
			return errors.New("invalid CLICKHOUSE_NODE_ID")
		}
	}
	s, err := newClickHouseLogStoreWithFallback(context.Background(), dsn, node, LOG_DB)
	if err != nil {
		return err
	}
	clickHouseLogs = s
	common.UsingClickHouse = true
	common.SysLog("using ClickHouse for logs")
	return nil
}

func (s *clickHouseLogStore) Close() error {
	return s.db.Close()
}

func createLog(ctx context.Context, log *Log) error {
	if clickHouseLogs != nil {
		return clickHouseLogs.Create(ctx, log)
	}
	return LOG_DB.Create(log).Error
}
