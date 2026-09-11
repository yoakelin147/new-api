package model

import (
	"context"
	"errors"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestLogWriteFailoverSwitchesAndRecovers(t *testing.T) {
	now := time.Now()
	primaryUp := true
	primaryCalls, backupCalls := 0, 0
	f := newLogWriteFailover(func(_ context.Context, log *Log) error {
		primaryCalls++
		if !primaryUp {
			return errors.New("ClickHouse unavailable")
		}
		return nil
	}, func(_ context.Context, log *Log) error { backupCalls++; return nil })
	f.now = func() time.Time { return now }
	require.NoError(t, f.Write(context.Background(), &Log{Content: "healthy"}))
	require.Equal(t, 0, backupCalls)
	primaryUp = false
	require.NoError(t, f.Write(context.Background(), &Log{Content: "outage"}))
	require.Equal(t, 1, backupCalls)
	require.NoError(t, f.Write(context.Background(), &Log{Content: "bypasses failing primary"}))
	require.Equal(t, 2, primaryCalls)
	require.Equal(t, 2, backupCalls)
	primaryUp = true
	now = now.Add(logFailoverRetryInterval)
	require.NoError(t, f.Write(context.Background(), &Log{Content: "recovered"}))
	require.NoError(t, f.Write(context.Background(), &Log{Content: "healthy again"}))
	require.Equal(t, 4, primaryCalls)
	require.Equal(t, 2, backupCalls)
}

func TestLogWriteFailoverReportsBothFailures(t *testing.T) {
	primaryErr, backupErr := errors.New("primary down"), errors.New("backup down")
	f := newLogWriteFailover(func(context.Context, *Log) error { return primaryErr }, func(context.Context, *Log) error { return backupErr })
	err := f.Write(context.Background(), &Log{})
	require.ErrorIs(t, err, primaryErr)
	require.ErrorIs(t, err, backupErr)
}

func TestLogWriteFailoverPersistsAfterPrimaryCancellation(t *testing.T) {
	f := newLogWriteFailover(func(ctx context.Context, _ *Log) error { return ctx.Err() }, func(ctx context.Context, _ *Log) error { return ctx.Err() })
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	require.NoError(t, f.Write(ctx, &Log{}))
}

func TestLogWriteFailoverLateSuccessDoesNotClearNewOutage(t *testing.T) {
	started, finish := make(chan struct{}), make(chan struct{})
	var primaryCalls atomic.Int32
	f := newLogWriteFailover(func(context.Context, *Log) error {
		if primaryCalls.Add(1) == 1 {
			close(started)
			<-finish
			return nil
		}
		return errors.New("down")
	}, func(context.Context, *Log) error { return nil })
	done := make(chan error, 1)
	go func() { done <- f.Write(context.Background(), &Log{}) }()
	<-started
	require.NoError(t, f.Write(context.Background(), &Log{}))
	close(finish)
	require.NoError(t, <-done)
	require.NoError(t, f.Write(context.Background(), &Log{}))
	require.Equal(t, int32(2), primaryCalls.Load())
}
