package model

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/QuantumNous/new-api/common"
	"gorm.io/gorm"
)

const logFailoverRetryInterval = 5 * time.Second
const logFailoverWriteTimeout = 5 * time.Second

// logWriteFailover protects log writes only. It never presents a partial backup
// database as the full log history, nor replays ambiguous writes automatically.
type logWriteFailover struct {
	primary, backup func(context.Context, *Log) error
	now             func() time.Time
	mu              sync.Mutex
	retryAt         time.Time
	generation      uint64
	lastError       error
}

func newLogWriteFailover(primary, backup func(context.Context, *Log) error) *logWriteFailover {
	return &logWriteFailover{primary: primary, backup: backup, now: time.Now}
}

func (f *logWriteFailover) begin() (bool, uint64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if !f.retryAt.IsZero() {
		if f.now().Before(f.retryAt) {
			return false, f.generation, f.lastError
		}
		// Permit just one recovery attempt while other writers use the original DB.
		f.retryAt = f.now().Add(logFailoverRetryInterval)
	}
	return true, f.generation, f.lastError
}

func (f *logWriteFailover) unavailable(err error) {
	f.mu.Lock()
	firstFailure := f.retryAt.IsZero()
	f.generation++
	f.lastError = err
	f.retryAt = f.now().Add(logFailoverRetryInterval)
	f.mu.Unlock()
	if firstFailure {
		common.SysError("ClickHouse log writes degraded to the original log database: " + err.Error())
	}
}

func (f *logWriteFailover) recovered(generation uint64) {
	f.mu.Lock()
	recovered := generation == f.generation && !f.retryAt.IsZero()
	if recovered {
		f.retryAt = time.Time{}
		f.lastError = nil
	}
	f.mu.Unlock()
	if recovered {
		common.SysLog("ClickHouse log writes recovered; outage logs remain in the original log database")
	}
}

func (f *logWriteFailover) Write(ctx context.Context, log *Log) error {
	attempt, generation, primaryErr := f.begin()
	if attempt {
		primaryCtx, cancel := context.WithTimeout(ctx, logFailoverWriteTimeout)
		primaryErr = f.primary(primaryCtx, log)
		cancel()
		if primaryErr == nil {
			f.recovered(generation)
			return nil
		}
		f.unavailable(primaryErr)
	}
	// A cancelled/expired primary call must not also cancel the disaster write.
	backupCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), logFailoverWriteTimeout)
	defer cancel()
	if err := f.backup(backupCtx, log); err != nil {
		return errors.Join(fmt.Errorf("ClickHouse log write unavailable: %w", primaryErr), fmt.Errorf("original log database write failed: %w", err))
	}
	return nil
}

func saveLogToOriginalDatabase(ctx context.Context, db *gorm.DB, log *Log) error {
	copy := *log
	// ClickHouse IDs must not advance the original SQL sequence or collide with
	// existing history. Let the original database generate its usual local ID.
	copy.Id = 0
	return db.WithContext(ctx).Create(&copy).Error
}
