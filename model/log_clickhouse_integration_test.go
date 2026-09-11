package model

import (
	"context"
	"fmt"
	"net/url"
	"os"
	"strconv"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/stretchr/testify/require"
)

func clickHouseTestDatabase(t *testing.T) (*clickHouseLogStore, string) {
	t.Helper()
	base := os.Getenv("TEST_CLICKHOUSE_DSN")
	if base == "" {
		t.Skip("set TEST_CLICKHOUSE_DSN for real database tests")
	}
	u, err := url.Parse(base)
	require.NoError(t, err)
	u.Path = fmt.Sprintf("/new_api_test_%d", time.Now().UnixNano())
	s, err := newClickHouseLogStore(context.Background(), u.String(), 15)
	require.NoError(t, err)
	t.Cleanup(func() {
		_, err := s.db.Exec("DROP DATABASE " + quoteClickHouseIdentifier(s.database) + " SYNC")
		require.NoError(t, err)
		s.Close()
	})
	return s, u.String()
}

func TestClickHouseScanExportAndDeleteBeyondOneBatch(t *testing.T) {
	s, _ := clickHouseTestDatabase(t)
	ctx := context.Background()
	tx, err := s.db.BeginTx(ctx, nil)
	require.NoError(t, err)
	defer tx.Rollback()
	batch, err := tx.PrepareContext(ctx, "INSERT INTO "+s.table()+" (id, user_id, created_at, type, username, token_id, token_name, model_name, quota, prompt_tokens, completion_tokens, other)")
	require.NoError(t, err)
	defer batch.Close()
	now := time.Now().Unix()
	const count = 1205
	for i := 1; i <= count; i++ {
		_, err = batch.ExecContext(ctx, int64(i), int64(10), now, int64(LogTypeConsume), "fixture", int64(i), "token", "model-"+strconv.Itoa(i), int64(1), int64(3), int64(4), "{}")
		require.NoError(t, err)
	}
	require.NoError(t, tx.Commit())
	old := clickHouseLogs
	clickHouseLogs = s
	t.Cleanup(func() { clickHouseLogs = old })
	rows, total, sum, err := GetCostStatistics(CostStatisticsQuery{IsAdmin: true, StartTimestamp: now - 1, EndTimestamp: now + 1, ExportAll: true}, 0, 0)
	require.NoError(t, err)
	require.Len(t, rows, count)
	require.Equal(t, count, total)
	require.Equal(t, CostStatisticsSummary{Quota: count, RequestCount: count, PromptTokens: count * 3, CompletionTokens: count * 4, TotalTokens: count * 7}, sum)
	profits, total, psum, err := GetProfitStatistics(ProfitStatisticsQuery{StartTimestamp: now - 1, EndTimestamp: now + 1, ExportAll: true}, 0, 0)
	require.NoError(t, err)
	require.Len(t, profits, count)
	require.Equal(t, count, total)
	require.Equal(t, count, psum.RequestCount)
	page, total64, err := s.Query(ctx, logFilter{UserIDs: []int{10}}, 1000, 100, logOrderID)
	require.NoError(t, err)
	require.Len(t, page, 100)
	require.Equal(t, int64(count), total64)
	deleted, err := DeleteOldLog(ctx, now+1, 100)
	require.NoError(t, err)
	require.Equal(t, int64(count), deleted)
	page, total64, err = s.Query(ctx, logFilter{}, 0, 10, logOrderID)
	require.NoError(t, err)
	require.Empty(t, page)
	require.Zero(t, total64)
}

func TestClickHouseExistingIncompatibleTableIsNotChanged(t *testing.T) {
	s, dsn := clickHouseTestDatabase(t)
	_, err := s.db.Exec("ALTER TABLE " + s.table() + " DROP COLUMN quota")
	require.NoError(t, err)
	var before, after string
	require.NoError(t, s.db.QueryRow("SHOW CREATE TABLE "+s.table()).Scan(&before))
	_, err = newClickHouseLogStore(context.Background(), dsn, 15)
	require.ErrorContains(t, err, "quota")
	require.NoError(t, s.db.QueryRow("SHOW CREATE TABLE "+s.table()).Scan(&after))
	require.Equal(t, before, after)
}

func TestInitLogDBClickHouseIsIndependentOfMainDatabase(t *testing.T) {
	_, dsn := clickHouseTestDatabase(t)
	t.Setenv("CLICKHOUSE_DSN", dsn)
	t.Setenv("CLICKHOUSE_NODE_ID", "20")
	oldStore, oldLogs, oldCH := clickHouseLogs, LOG_DB, common.UsingClickHouse
	oldMySQL, oldPostgres := common.UsingMySQL, common.UsingPostgreSQL
	t.Cleanup(func() {
		clickHouseLogs = oldStore
		LOG_DB = oldLogs
		common.UsingClickHouse = oldCH
		common.UsingMySQL = oldMySQL
		common.UsingPostgreSQL = oldPostgres
	})
	for _, mainType := range []string{"mysql", "postgres"} {
		t.Run(mainType, func(t *testing.T) {
			common.UsingMySQL, common.UsingPostgreSQL = mainType == "mysql", mainType == "postgres"
			mainDB := DB
			require.NoError(t, InitLogDB())
			require.Same(t, mainDB, DB)
			require.True(t, common.UsingClickHouse)
			require.NotNil(t, clickHouseLogs)
			require.NoError(t, clickHouseLogs.Close())
			clickHouseLogs = nil
		})
	}
}

func TestInitLogDBEmptyClickHouseKeepsOriginalDatabase(t *testing.T) {
	t.Setenv("CLICKHOUSE_DSN", " \t ")
	t.Setenv("LOG_SQL_DSN", "")
	t.Setenv("CLICKHOUSE_NODE_ID", "unused-invalid-value")
	oldLogs, oldStore := LOG_DB, clickHouseLogs
	clickHouseLogs = nil
	t.Cleanup(func() { LOG_DB = oldLogs; clickHouseLogs = oldStore })
	require.NoError(t, InitLogDB())
	require.Same(t, DB, LOG_DB)
	require.Nil(t, clickHouseLogs)
}
