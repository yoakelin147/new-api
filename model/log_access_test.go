package model

import (
	"context"
	"testing"

	"github.com/glebarez/sqlite"
	"github.com/stretchr/testify/require"
	"gorm.io/gorm"
)

func TestUnifiedLogAccessUsesOriginalDatabaseAndPreservesContext(t *testing.T) {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&Log{}))
	require.NoError(t, db.Create(&[]Log{
		{UserId: 7, CreatedAt: 100, Type: LogTypeConsume, ChannelId: 3, ModelName: "wanted", RequestId: "one"},
		{UserId: 8, CreatedAt: 101, Type: LogTypeConsume, ChannelId: 4, ModelName: "other", RequestId: "two"},
	}).Error)

	oldLogDB, oldClickHouse := LOG_DB, clickHouseLogs
	LOG_DB, clickHouseLogs = db, nil
	t.Cleanup(func() {
		LOG_DB, clickHouseLogs = oldLogDB, oldClickHouse
		_ = closeDB(db)
	})

	state := NewLogReadState(context.Background())
	filter := logFilter{UserIDs: []int{7}, ModelName: "wanted"}
	logs, total, err := queryLogs(state, filter, 0, 10, logOrderID)
	require.NoError(t, err)
	require.Equal(t, int64(1), total)
	require.Equal(t, "one", logs[0].RequestId)
	require.Equal(t, "sql", state.Source())

	var visited []string
	require.NoError(t, forEachLogs(state, filter, func(log Log) error {
		visited = append(visited, log.RequestId)
		return nil
	}))
	require.Equal(t, []string{"one"}, visited)

	ids, err := listLogChannelIDs(state, filter)
	require.NoError(t, err)
	require.Equal(t, map[int]struct{}{3: {}}, ids)
	_, _, err = queryLogs(state, filter, -1, 10, logOrderID)
	require.ErrorContains(t, err, "pagination")
	_, _, err = queryLogs(state, filter, 0, 10, logOrder("unsupported"))
	require.ErrorContains(t, err, "order")

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, _, err = queryLogs(NewLogReadState(ctx), logFilter{}, 0, 10, logOrderID)
	require.ErrorIs(t, err, context.Canceled)
}
