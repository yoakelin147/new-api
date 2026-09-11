package model

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestLogSQLFallbackPreservesFieldsAndExistingHistory(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	t.Cleanup(func() { closeDB(db) })
	require.NoError(t, db.AutoMigrate(&Log{}))
	history := Log{Id: 999, Content: "existing history"}
	require.NoError(t, db.Create(&history).Error)
	log := Log{Id: 999, UserId: 7, CreatedAt: 100, Type: LogTypeConsume, Content: "中文", Username: "user", TokenName: "key", ModelName: "model", Quota: 321, PromptTokens: 20, CompletionTokens: 5, UseTime: 2, IsStream: true, ChannelId: 3, TokenId: 4, Group: "default", Ip: "127.0.0.1", RequestId: "request", UpstreamRequestId: "upstream", Other: `{"unknown":{"value":0}}`}
	want := log
	require.NoError(t, saveLogToOriginalDatabase(context.Background(), db, &log))
	require.Equal(t, want, log)
	var saved []Log
	require.NoError(t, db.Order("id").Find(&saved).Error)
	want.Id = 1000
	require.Equal(t, []Log{history, want}, saved)
}
