package model

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/mysql"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func TestClickHouseLogStoreIntegration(t *testing.T) {
	dsn := os.Getenv("TEST_CLICKHOUSE_DSN")
	if dsn == "" {
		t.Skip("set TEST_CLICKHOUSE_DSN to run against a real ClickHouse server")
	}
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	u.Path = fmt.Sprintf("/new_api_test_%d", time.Now().UnixNano())
	ctx := context.Background()
	s, err := newClickHouseLogStore(ctx, u.String(), 7)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := s.db.ExecContext(ctx, "DROP DATABASE "+quoteClickHouseIdentifier(s.database)+" SYNC")
		require.NoError(t, err)
		require.NoError(t, s.Close())
	})
	now := time.Now().Unix()
	want := &Log{UserId: 10, CreatedAt: now, Type: LogTypeConsume, Content: "中文 ' \\", Username: "alice", TokenName: "token", ModelName: "model_%", Quota: 100, PromptTokens: 20, CompletionTokens: 5, UseTime: 3, IsStream: true, ChannelId: 18, TokenId: 8, Group: "default", Ip: "127.0.0.1", RequestId: "same-request", UpstreamRequestId: "upstream", Other: `{"token_audit":{"masked_key":"sk-ab***cd"},"stream_status":{"status":"error","end_reason":"client_gone"}}`}
	require.NoError(t, s.Create(ctx, want))
	logs, total, err := s.Query(ctx, logFilter{UserIDs: []int{10}}, 0, 10, logOrderCreated)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Equal(t, []*Log{want}, logs)
	empty, total, err := s.Query(ctx, logFilter{UserIDs: []int{}}, 0, 10, logOrderID)
	require.NoError(t, err)
	require.Empty(t, empty)
	require.Zero(t, total)
	logs, _, err = s.Query(ctx, logFilter{ModelName: "_%", ModelContains: true, StreamStatus: StreamStatusFilterClientGone}, 0, 10, logOrderID)
	require.NoError(t, err)
	require.Equal(t, []*Log{want}, logs)
	var before string
	require.NoError(t, s.db.QueryRowContext(ctx, "SHOW CREATE TABLE "+s.table()).Scan(&before))
	reopened, err := newClickHouseLogStore(ctx, u.String(), 7)
	require.NoError(t, err)
	defer reopened.Close()
	var after string
	require.NoError(t, reopened.db.QueryRowContext(ctx, "SHOW CREATE TABLE "+s.table()).Scan(&after))
	require.Equal(t, before, after, "existing table definition must not be migrated")
	refund := &Log{UserId: 10, CreatedAt: now + 1, Type: LogTypeRefund, Quota: 40, Other: "broken json"}
	require.NoError(t, reopened.Create(ctx, refund))
	require.Greater(t, refund.Id, want.Id)
	count, err := s.DeleteBefore(ctx, now+1)
	require.NoError(t, err)
	require.Equal(t, int64(1), count)
	logs, total, err = s.Query(ctx, logFilter{}, 0, 10, logOrderID)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Equal(t, []*Log{refund}, logs)
}

func TestClickHouseModelParity(t *testing.T) {
	dsn := os.Getenv("TEST_CLICKHOUSE_DSN")
	if dsn == "" {
		t.Skip("set TEST_CLICKHOUSE_DSN for database parity tests")
	}
	withClickHouseParityPrimary(t)
	oldStore, oldLogDB, oldCache := clickHouseLogs, LOG_DB, common.MemoryCacheEnabled
	common.MemoryCacheEnabled = false
	t.Cleanup(func() { clickHouseLogs = oldStore; LOG_DB = oldLogDB; common.MemoryCacheEnabled = oldCache })
	u, err := url.Parse(dsn)
	require.NoError(t, err)
	u.Path = fmt.Sprintf("/new_api_test_%d", time.Now().UnixNano())
	s, err := newClickHouseLogStore(context.Background(), u.String(), 12)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := s.db.Exec("DROP DATABASE " + quoteClickHouseIdentifier(s.database) + " SYNC")
		require.NoError(t, err)
		s.Close()
	})
	require.NoError(t, DB.Create(&Channel{Id: 18, Name: "test channel", Key: "test"}).Error)
	require.NoError(t, DB.Create(&ChannelProfitRatio{ChannelId: 18, Enabled: true, UpstreamRatio: 0.3}).Error)
	now := time.Now().Unix()
	logs := []*Log{
		{UserId: 10, CreatedAt: now - 10, Type: LogTypeConsume, Username: "alice", TokenName: "key", ModelName: "gpt-test", TokenId: 58, ChannelId: 18, Quota: 500, PromptTokens: 100, CompletionTokens: 20, UseTime: 2, Group: "default", Other: `{"group_ratio":0.5,"model_ratio":1,"completion_ratio":2,"cache_tokens":40,"cache_ratio":0.1,"token_audit":{"masked_key":"sk-a***b"}}`},
		{UserId: 10, CreatedAt: now - 5, Type: LogTypeConsume, Username: "alice", TokenName: "key", ModelName: "claude-test", TokenId: 58, ChannelId: 18, Quota: 800, PromptTokens: 10, CompletionTokens: 5, Group: "default", Other: `{"usage_semantic":"anthropic","group_ratio":0.5,"cache_tokens":80,"cache_write_tokens":10,"stream_status":{"status":"error","end_reason":"client_gone"}}`},
		{UserId: 10, CreatedAt: now - 3, Type: LogTypeRefund, Username: "alice", TokenName: "key", ModelName: "gpt-test", TokenId: 58, ChannelId: 18, Quota: 200, Group: "default", Other: `{"group_ratio":0.5}`},
		{UserId: 20, CreatedAt: now - 2, Type: LogTypeConsume, Username: "bob", TokenName: "second", ModelName: "gpt-test", TokenId: 59, ChannelId: 18, Quota: 100, PromptTokens: 5, Group: "other", Other: "bad json"},
	}
	require.NoError(t, LOG_DB.Create(logs).Error)
	for _, log := range logs {
		require.NoError(t, s.Create(context.Background(), log))
	}
	read := func() map[string]any {
		m := map[string]any{}
		all, total, err := GetAllLogs(0, now-100, now, "", "", "", 0, 10, 0, "", "", "", "")
		require.NoError(t, err)
		m["all"], m["total"] = all, total
		self, _, err := GetUserLogs(10, 0, now-100, now, "", "", 0, 10, "", "", "")
		require.NoError(t, err)
		m["self"] = self
		managed, _, err := GetManagedUserLogs([]int{10}, 0, now-100, now, "", "", 0, 10, "", "", "")
		require.NoError(t, err)
		m["managed"] = managed
		stat, err := SumUsedQuota(0, now-100, now, "", "", "", 0, "", "")
		require.NoError(t, err)
		m["stat"] = stat
		cost, n, sum, err := GetCostStatistics(CostStatisticsQuery{IsAdmin: true, StartTimestamp: now - 100, EndTimestamp: now}, 0, 20)
		require.NoError(t, err)
		m["cost"], m["cost-total"], m["cost-summary"] = cost, n, sum
		profit, n, psum, err := GetProfitStatistics(ProfitStatisticsQuery{StartTimestamp: now - 100, EndTimestamp: now}, 0, 20)
		require.NoError(t, err)
		m["profit"], m["profit-total"], m["profit-summary"] = profit, n, psum
		channels, err := GetQuotaDataGroupByChannel(now-100, now)
		require.NoError(t, err)
		m["channels"] = channels
		tokens, err := GetQuotaDataGroupByToken(10, now-100, now)
		require.NoError(t, err)
		m["tokens"] = tokens
		users, err := GetQuotaDataGroupByUserToken(10, now-100, now)
		require.NoError(t, err)
		m["users"] = users
		cache, err := GetTokenCacheHitStats(TokenCacheHitStatsQuery{StartTimestamp: now - 100, EndTimestamp: now})
		require.NoError(t, err)
		m["cache"] = cache
		groups, err := GetTokenCacheHitGroupStats(TokenCacheHitStatsQuery{StartTimestamp: now - 100, EndTimestamp: now})
		require.NoError(t, err)
		m["groups"] = groups
		trend, err := GetTokenCacheHitTrendStats(TokenCacheHitStatsQuery{StartTimestamp: now - 100, EndTimestamp: now})
		require.NoError(t, err)
		m["trend"] = trend
		return m
	}
	want := read()
	// An empty old log database proves every migrated reader uses ClickHouse.
	empty, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, empty.AutoMigrate(&Log{}))
	defer func() { db, _ := empty.DB(); db.Close() }()
	LOG_DB, clickHouseLogs = empty, s
	got := read()
	// The existing Go profit accumulator iterates a map: its floating-point
	// summary can differ by a final bit even between two SQL-only executions.
	wantSummary := want["profit-summary"].(ProfitStatisticsSummary)
	gotSummary := got["profit-summary"].(ProfitStatisticsSummary)
	for _, pair := range [][2]float64{{wantSummary.CustomerAmount, gotSummary.CustomerAmount}, {wantSummary.OriginalAmount, gotSummary.OriginalAmount}, {wantSummary.UpstreamCost, gotSummary.UpstreamCost}, {wantSummary.Profit, gotSummary.Profit}, {wantSummary.ProfitRate, gotSummary.ProfitRate}, {wantSummary.UnconfiguredAmount, gotSummary.UnconfiguredAmount}} {
		require.InDelta(t, pair[0], pair[1], 1e-12)
	}
	gotSummary.CustomerAmount, gotSummary.OriginalAmount, gotSummary.UpstreamCost = wantSummary.CustomerAmount, wantSummary.OriginalAmount, wantSummary.UpstreamCost
	gotSummary.Profit, gotSummary.ProfitRate, gotSummary.UnconfiguredAmount = wantSummary.Profit, wantSummary.ProfitRate, wantSummary.UnconfiguredAmount
	got["profit-summary"] = gotSummary
	require.Equal(t, want, got)
}

func withClickHouseParityPrimary(t *testing.T) {
	t.Helper()
	dsn := os.Getenv("TEST_LOG_PRIMARY_DSN")
	if dsn == "" {
		truncateTables(t)
		return
	}
	// This optional test owns only these three tables in an explicitly named
	// test database. Never accept a production database as its target.
	if !strings.Contains(dsn, "/new_api_test") {
		t.Fatal("TEST_LOG_PRIMARY_DSN must target new_api_test")
	}
	var dialect gorm.Dialector
	isPostgres := strings.HasPrefix(dsn, "postgres")
	if isPostgres {
		dialect = postgres.Open(dsn)
	} else {
		dialect = mysql.Open(dsn)
	}
	db, err := gorm.Open(dialect, &gorm.Config{})
	require.NoError(t, err)
	require.Equal(t, "new_api_test", db.Migrator().CurrentDatabase())
	oldDB, oldLogs := DB, LOG_DB
	oldMySQL, oldPostgres, oldSQLite := common.UsingMySQL, common.UsingPostgreSQL, common.UsingSQLite
	DB, LOG_DB = db, db
	common.UsingMySQL, common.UsingPostgreSQL, common.UsingSQLite = !isPostgres, isPostgres, false
	t.Setenv("LOG_SQL_DSN", "")
	initCol()
	t.Cleanup(func() {
		require.NoError(t, db.Migrator().DropTable(&Log{}, &ChannelProfitRatio{}, &Channel{}))
		conn, _ := db.DB()
		conn.Close()
		DB, LOG_DB = oldDB, oldLogs
		common.UsingMySQL, common.UsingPostgreSQL, common.UsingSQLite = oldMySQL, oldPostgres, oldSQLite
		initCol()
	})
	require.NoError(t, db.AutoMigrate(&Log{}, &Channel{}, &ChannelProfitRatio{}))
}

func TestClickHouseDSNValidationDoesNotExposePassword(t *testing.T) {
	_, err := newClickHouseLogStore(context.Background(), "clickhouse://user:very-secret@localhost:bad/logs", 0)
	require.Error(t, err)
	require.NotContains(t, err.Error(), "very-secret")
}
