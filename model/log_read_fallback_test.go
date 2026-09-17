package model

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/require"
)

func TestLogReadFallbackKeepsFiltersAndUserScope(t *testing.T) {
	db := failoverSQLTestDB(t)
	want := Log{UserId: 7, CreatedAt: 100, Type: LogTypeConsume, Username: "alice", TokenId: 9, ModelName: "model_%", ChannelId: 3, Group: "default", RequestId: "request", Other: `{"stream_status":{"status":"error","end_reason":"client_gone"}}`}
	require.NoError(t, db.Create(&want).Error)
	other := want
	other.Id, other.UserId = 0, 8
	require.NoError(t, db.Create(&other).Error)
	s, err := openClickHouseLogStore("clickhouse://127.0.0.1:1/default", 0)
	require.NoError(t, err)
	defer s.Close()
	s.fallback = db
	state := &LogReadState{}
	filter := logFilter{UserIDs: []int{7}, Types: []int{LogTypeConsume}, Start: 99, End: 101, Group: "default", ModelName: "_%", ModelContains: true, RequestID: "request", StreamStatus: StreamStatusFilterClientGone, HasToken: true}
	rows, total, err := s.Query(logReadContext(state), filter, 0, 10, logOrderCreated)
	require.NoError(t, err)
	require.Equal(t, []*Log{&want}, rows)
	require.Equal(t, int64(1), total)
	require.Equal(t, "original", state.Source())
	var scanned []Log
	require.NoError(t, s.Scan(logReadContext(state), filter, func(log Log) error { scanned = append(scanned, log); return nil }))
	require.Equal(t, []Log{want}, scanned)
	ids, err := s.ChannelIDs(logReadContext(state), filter)
	require.NoError(t, err)
	require.Equal(t, map[int]struct{}{3: {}}, ids)
	filter.UserIDs = []int{}
	rows, total, err = s.Query(logReadContext(state), filter, 0, 10, logOrderID)
	require.NoError(t, err)
	require.Empty(t, rows)
	require.Zero(t, total)
	require.NoError(t, closeDB(db))
	_, _, err = s.Query(logReadContext(), logFilter{}, 0, 10, logOrderID)
	require.ErrorIs(t, err, errLogReadUnavailable)
	require.NotContains(t, err.Error(), "127.0.0.1")
	require.NotContains(t, err.Error(), "database is closed")
}

func TestLogReadFallbackDoesNotMixPartiallyConsumedData(t *testing.T) {
	s := &clickHouseLogStore{fallback: DB}
	state := &LogReadState{}
	count := 0
	err := s.readLogs(logReadContext(state), func(ctx context.Context) error {
		ctx.Value(logReadStateKey{}).(*LogReadState).source = "clickhouse"
		count++
		return errors.New("connection lost after reading the first row")
	}, func(context.Context) error {
		count += 100
		return nil
	})
	require.ErrorIs(t, err, errLogReadUnavailable)
	require.Equal(t, 1, count)
}

func TestLogReadFallbackPreservesCanceledContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	s := &clickHouseLogStore{fallback: DB}
	fallbackCalled := false

	err := s.readLogs(logReadContext(NewLogReadState(ctx)), func(ctx context.Context) error {
		return ctx.Err()
	}, func(context.Context) error {
		fallbackCalled = true
		return nil
	})

	require.ErrorIs(t, err, context.Canceled)
	require.False(t, fallbackCalled)
}

func TestDegradedLogReadNeverProbesClickHouse(t *testing.T) {
	s := &clickHouseLogStore{fallback: DB}
	s.readDegraded.Store(true)
	var primaryCalls atomic.Int32
	fallbackCalls := 0

	err := s.readLogs(logReadContext(), func(context.Context) error {
		primaryCalls.Add(1)
		return errors.New("business request probed ClickHouse")
	}, func(context.Context) error {
		fallbackCalls++
		return nil
	})

	require.NoError(t, err)
	require.Zero(t, primaryCalls.Load())
	require.Equal(t, 1, fallbackCalls)
}

func TestLogReadRecoveryProbeClearsDegradedState(t *testing.T) {
	db := failoverSQLTestDB(t)
	sqlDB, err := db.DB()
	require.NoError(t, err)
	s := &clickHouseLogStore{db: sqlDB, database: "main"}
	s.readDegraded.Store(true)

	s.probeReadRecovery(context.Background())

	require.False(t, s.readDegraded.Load())
}
