package model

import (
	"fmt"
	"testing"
	"time"

	"github.com/QuantumNous/new-api/common"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func withLogStreamFilterTestDatabase(t *testing.T) {
	t.Helper()
	previousLogDB := LOG_DB
	dsn := fmt.Sprintf("file:log-stream-filter-%d?mode=memory&cache=shared", time.Now().UnixNano())
	testDB, err := gorm.Open(sqlite.Open(dsn), &gorm.Config{})
	require.NoError(t, err)
	LOG_DB = testDB
	require.NoError(t, LOG_DB.AutoMigrate(&Log{}))
	t.Cleanup(func() { LOG_DB = previousLogDB })
}

func TestGetAllLogsFiltersStreamErrors(t *testing.T) {
	withLogStreamFilterTestDatabase(t)

	logs := []Log{
		{
			CreatedAt: time.Now().Unix(), Type: LogTypeConsume, Quota: 120,
			Other: common.MapToJsonStr(map[string]interface{}{
				"stream_status": map[string]interface{}{"status": "error", "end_reason": "client_gone"},
			}),
		},
		{
			CreatedAt: time.Now().Unix(), Type: LogTypeConsume, Quota: 80,
			Other: common.MapToJsonStr(map[string]interface{}{
				"stream_status": map[string]interface{}{"status": "error", "end_reason": "timeout"},
			}),
		},
		{
			CreatedAt: time.Now().Unix(), Type: LogTypeConsume, Quota: 60,
			Other: common.MapToJsonStr(map[string]interface{}{
				"stream_status": map[string]interface{}{"status": "ok", "end_reason": "done"},
			}),
		},
		{
			CreatedAt: time.Now().Unix(), Type: LogTypeError, Quota: 40,
			Other: common.MapToJsonStr(map[string]interface{}{
				"audit_info": map[string]interface{}{"status": "error"},
			}),
		},
	}
	require.NoError(t, LOG_DB.Create(&logs).Error)

	results, total, err := GetAllLogs(LogTypeUnknown, 0, 0, "", "", "", 0, 20, 0, "", "", "", StreamStatusFilterError)
	require.NoError(t, err)
	require.Equal(t, int64(2), total)
	require.Len(t, results, 2)

	results, total, err = GetAllLogs(LogTypeUnknown, 0, 0, "", "", "", 0, 20, 0, "", "", "", StreamStatusFilterClientGone)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Len(t, results, 1)
	require.Equal(t, 120, results[0].Quota)
}

func TestSumUsedQuotaFiltersStreamErrors(t *testing.T) {
	withLogStreamFilterTestDatabase(t)

	logs := []Log{
		{
			CreatedAt: time.Now().Unix(), Type: LogTypeConsume, Quota: 120, PromptTokens: 10, CompletionTokens: 5,
			Other: common.MapToJsonStr(map[string]interface{}{
				"stream_status": map[string]interface{}{"status": "error", "end_reason": "client_gone"},
			}),
		},
		{
			CreatedAt: time.Now().Unix(), Type: LogTypeConsume, Quota: 60, PromptTokens: 20, CompletionTokens: 10,
			Other: common.MapToJsonStr(map[string]interface{}{
				"stream_status": map[string]interface{}{"status": "ok", "end_reason": "done"},
			}),
		},
	}
	require.NoError(t, LOG_DB.Create(&logs).Error)

	stat, err := SumUsedQuota(LogTypeUnknown, 0, 0, "", "", "", 0, "", StreamStatusFilterError)
	require.NoError(t, err)
	require.Equal(t, 120, stat.Quota)
	require.Equal(t, 1, stat.Rpm)
	require.Equal(t, 15, stat.Tpm)
}
