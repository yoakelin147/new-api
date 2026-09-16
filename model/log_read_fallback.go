package model

import (
	"context"
	"database/sql"
	"errors"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
	"github.com/QuantumNous/new-api/pkg/clickhousealert"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// LogReadState keeps a multi-query response on one source and reports whether
// its records came from the original database, which may lack recent history.
// One request owns this state; it must not be shared between requests.
type LogReadState struct {
	source string
}

func (s *LogReadState) Source() string { return s.source }

type logReadStateKey struct{}

func logReadContext(states ...*LogReadState) context.Context {
	state := &LogReadState{}
	if len(states) > 0 && states[0] != nil {
		state = states[0]
	}
	return context.WithValue(context.Background(), logReadStateKey{}, state)
}

var errLogReadUnavailable = errors.New("日志暂时无法查询，请稍后重试")

func (s *clickHouseLogStore) readLogs(ctx context.Context, primary, fallback func(context.Context) error) error {
	state, _ := ctx.Value(logReadStateKey{}).(*LogReadState)
	if state == nil {
		state = &LogReadState{}
		ctx = context.WithValue(ctx, logReadStateKey{}, state)
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	attempt := state.source == "clickhouse" || (state.source == "" && !s.readDegraded.Load())
	if attempt {
		if err := primary(ctx); err == nil {
			state.source = "clickhouse"
			return nil
		} else {
			if s.fallback == nil {
				return err
			}
			if s.readDegraded.CompareAndSwap(false, true) {
				common.SysError("ClickHouse log reads degraded to the original database: " + err.Error())
				clickhousealert.Notify(clickhousealert.EventReadDegraded, "ClickHouse 读取失败或超时")
			}
			// A streaming consumer or earlier query already used this source.
			// Abort instead of mixing partial ClickHouse data with SQL aggregates.
			if state.source == "clickhouse" || s.fallback == nil || ctx.Err() != nil {
				return errLogReadUnavailable
			}
		}
	}
	if s.fallback == nil {
		return errLogReadUnavailable
	}
	state.source = "original"
	if err := fallback(ctx); err != nil {
		common.SysError("original log database query failed: " + err.Error())
		return errLogReadUnavailable
	}
	return nil
}

func (s *clickHouseLogStore) probeReadRecovery(ctx context.Context) {
	if !s.readProbeRunning.CompareAndSwap(false, true) {
		return
	}
	defer s.readProbeRunning.Store(false)

	var ready int
	err := s.db.QueryRowContext(ctx, "SELECT 1 FROM "+s.table()+" LIMIT 1").Scan(&ready)
	if errors.Is(err, sql.ErrNoRows) {
		err = nil
	}
	if err == nil && s.readDegraded.CompareAndSwap(true, false) {
		common.SysLog("ClickHouse log reads recovered; new queries use ClickHouse")
		clickhousealert.Notify(clickhousealert.EventReadRecovered, "ClickHouse 读取恢复")
	}
}

func (f logFilter) sqlQuery(db *gorm.DB) (*gorm.DB, error) {
	tx := db.Model(&Log{})
	if f.Types != nil {
		tx = tx.Where("type IN ?", f.Types)
	}
	if f.UserIDs != nil {
		// Empty scopes must never become an unfiltered query.
		tx = tx.Where("user_id IN ?", f.UserIDs)
	}
	if f.TokenID != nil {
		tx = tx.Where("token_id = ?", *f.TokenID)
	}
	if f.Start != 0 {
		tx = tx.Where("created_at >= ?", f.Start)
	}
	if f.End != 0 {
		tx = tx.Where("created_at <= ?", f.End)
	}
	for _, text := range []struct {
		column, value string
		contains      bool
	}{{"username", f.Username, false}, {"model_name", f.ModelName, f.ModelContains}} {
		if text.value == "" {
			continue
		}
		if f.ExactText {
			tx = tx.Where(clause.Eq{Column: text.column, Value: text.value})
		} else if text.contains {
			pattern := strings.NewReplacer("!", "!!", "%", "!%", "_", "!_").Replace(text.value)
			tx = tx.Where(text.column+" LIKE ? ESCAPE '!'", "%"+pattern+"%")
		} else {
			var err error
			tx, err = applyExplicitLogTextFilter(tx, text.column, text.value)
			if err != nil {
				return nil, err
			}
		}
	}
	for _, text := range []struct{ column, value string }{{"token_name", f.TokenName}, {"group", f.Group}, {"request_id", f.RequestID}, {"upstream_request_id", f.UpstreamRequestID}} {
		if text.value != "" {
			tx = tx.Where(clause.Eq{Column: text.column, Value: text.value})
		}
	}
	if f.ChannelID != 0 {
		tx = tx.Where("channel_id = ?", f.ChannelID)
	}
	if f.NonzeroChannel {
		tx = tx.Where("channel_id <> 0")
	}
	if f.NonzeroUser {
		tx = tx.Where("user_id <> 0")
	}
	if f.HasToken {
		tx = tx.Where("token_id <> 0 OR token_name <> ''")
	}
	return applyStreamStatusFilter(tx, f.StreamStatus)
}
