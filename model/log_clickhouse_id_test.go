package model

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestClickHouseLogIDsSurviveRestartAndClockRollback(t *testing.T) {
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	clock := func() time.Time { return now }
	g, err := newLogIDGenerator(7, 123456, clock)
	require.NoError(t, err)
	first, err := g.Next(context.Background())
	require.NoError(t, err)
	require.Greater(t, first, int64(123456))
	require.LessOrEqual(t, first, int64(1<<53-1))
	restarted, err := newLogIDGenerator(7, first, clock)
	require.NoError(t, err)
	now = now.Add(-time.Second)
	second, err := restarted.Next(context.Background())
	require.NoError(t, err)
	require.Greater(t, second, first)
	other, err := newLogIDGenerator(8, second, clock)
	require.NoError(t, err)
	third, err := other.Next(context.Background())
	require.NoError(t, err)
	require.NotEqual(t, second, third)
}

func TestClickHouseLogIDExhaustionIsCancellable(t *testing.T) {
	now := time.Date(2026, 9, 11, 12, 0, 0, 0, time.UTC)
	g, err := newLogIDGenerator(0, 0, func() time.Time { return now })
	require.NoError(t, err)
	seen := make(map[int64]bool)
	for i := 0; i < 4096; i++ {
		id, err := g.Next(context.Background())
		require.NoError(t, err)
		require.False(t, seen[id])
		seen[id] = true
	}
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	_, err = g.Next(ctx)
	require.ErrorIs(t, err, context.Canceled)
}

func TestClickHouseLogIDRejectsInvalidNodeAndUnsafeHistory(t *testing.T) {
	for _, node := range []int{-1, 1024} {
		_, err := newLogIDGenerator(node, 0, time.Now)
		require.Error(t, err)
	}
	_, err := newLogIDGenerator(0, 1<<53, time.Now)
	require.Error(t, err)
}
