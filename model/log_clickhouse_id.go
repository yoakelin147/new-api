package model

import (
	"context"
	"errors"
	"sync"
	"time"
)

const logIDEpoch int64 = 1577836800 // 2020-01-01 UTC
const maxLogID int64 = 1<<53 - 1    // Exact in existing JSON/JavaScript clients.

// logIDGenerator uses 31 seconds bits, 10 explicitly assigned writer bits and
// 12 sequence bits. Each concurrently running New API writer needs a distinct
// node ID. Seeding above the stored maximum protects restarts and imported IDs.
type logIDGenerator struct {
	mu       sync.Mutex
	node     int64
	tick     int64
	sequence int64
	now      func() time.Time
}

func newLogIDGenerator(node int, storedMax int64, now func() time.Time) (*logIDGenerator, error) {
	if node < 0 || node > 1023 {
		return nil, errors.New("CLICKHOUSE_NODE_ID must be between 0 and 1023")
	}
	if storedMax < 0 || storedMax > maxLogID {
		return nil, errors.New("ClickHouse log IDs exceed the existing JSON integer range")
	}
	tick := now().Unix() - logIDEpoch
	if storedMax>>22 >= tick {
		tick = (storedMax >> 22) + 1
	}
	if tick < 0 || tick >= 1<<31 {
		return nil, errors.New("ClickHouse log ID timestamp is out of range")
	}
	return &logIDGenerator{node: int64(node), tick: tick, sequence: -1, now: now}, nil
}

func (g *logIDGenerator) Next(ctx context.Context) (int64, error) {
	for {
		if err := ctx.Err(); err != nil {
			return 0, err
		}
		g.mu.Lock()
		tick := g.now().Unix() - logIDEpoch
		if tick > g.tick {
			g.tick, g.sequence = tick, -1
		}
		if g.tick >= 1<<31 {
			g.mu.Unlock()
			return 0, errors.New("ClickHouse log ID timestamp is out of range")
		}
		if g.sequence < 4095 {
			g.sequence++
			id := g.tick<<22 | g.node<<12 | g.sequence
			g.mu.Unlock()
			return id, nil
		}
		g.mu.Unlock()
		timer := time.NewTimer(time.Millisecond)
		select {
		case <-ctx.Done():
			timer.Stop()
			return 0, ctx.Err()
		case <-timer.C:
		}
	}
}
