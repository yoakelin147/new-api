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
	ctx    context.Context
}

func (s *LogReadState) Source() string { return s.source }

func NewLogReadState(ctx context.Context) *LogReadState {
	if ctx == nil {
		ctx = context.Background()
	}
	return &LogReadState{ctx: ctx}
}

func firstLogReadState(states ...*LogReadState) *LogReadState {
	if len(states) > 0 && states[0] != nil {
		return states[0]
	}
	return NewLogReadState(context.Background())
}

type logReadStateKey struct{}

func logReadContext(states ...*LogReadState) context.Context {
	state := firstLogReadState(states...)
	if state.ctx == nil {
		state.ctx = context.Background()
	}
	return context.WithValue(state.ctx, logReadStateKey{}, state)
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
			if ctx.Err() != nil {
				return ctx.Err()
			}
			if s.fallback == nil {
				return err
			}
			if s.readDegraded.CompareAndSwap(false, true) {
				common.SysError("ClickHouse log reads degraded to the original database: " + err.Error())
				clickhousealert.Notify(clickhousealert.EventReadDegraded, "ClickHouse 读取失败或超时")
			}
			// A streaming consumer or earlier query already used this source.
			// Abort instead of mixing partial ClickHouse data with SQL aggregates.
			if state.source == "clickhouse" || s.fallback == nil {
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

func queryLogs(state *LogReadState, filter logFilter, offset, limit int, order logOrder) ([]*Log, int64, error) {
	if offset < 0 || limit <= 0 {
		return nil, 0, errors.New("invalid log pagination")
	}
	if order != logOrderID && order != logOrderCreated {
		return nil, 0, errors.New("invalid log order")
	}
	state = firstLogReadState(state)
	ctx := logReadContext(state)
	if clickHouseLogs != nil {
		return clickHouseLogs.Query(ctx, filter, offset, limit, order)
	}
	state.source = "sql"
	return querySQLLogs(ctx, LOG_DB, filter, offset, limit, order)
}

func forEachLogs(state *LogReadState, filter logFilter, visit func(Log) error) error {
	state = firstLogReadState(state)
	ctx := logReadContext(state)
	if clickHouseLogs != nil {
		return clickHouseLogs.Scan(ctx, filter, visit)
	}
	state.source = "sql"
	return scanFilteredSQLLogs(ctx, LOG_DB, filter, visit)
}

func listLogChannelIDs(state *LogReadState, filter logFilter) (map[int]struct{}, error) {
	state = firstLogReadState(state)
	ctx := logReadContext(state)
	if clickHouseLogs != nil {
		return clickHouseLogs.ChannelIDs(ctx, filter)
	}
	state.source = "sql"
	return listSQLLogChannelIDs(ctx, LOG_DB, filter)
}

func querySQLLogs(ctx context.Context, db *gorm.DB, filter logFilter, offset, limit int, order logOrder) ([]*Log, int64, error) {
	tx, err := filter.sqlQuery(db.WithContext(ctx))
	if err != nil {
		return nil, 0, err
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	logs := make([]*Log, 0)
	err = tx.Order(string(order)).Offset(offset).Limit(limit).Find(&logs).Error
	return logs, total, err
}

func scanFilteredSQLLogs(ctx context.Context, db *gorm.DB, filter logFilter, visit func(Log) error) error {
	tx, err := filter.sqlQuery(db.WithContext(ctx))
	if err != nil {
		return err
	}
	return scanSQLLogs(tx, visit)
}

func scanSQLLogs(tx *gorm.DB, visit func(Log) error) error {
	var batch []Log
	return tx.FindInBatches(&batch, 1000, func(_ *gorm.DB, _ int) error {
		for _, log := range batch {
			if err := visit(log); err != nil {
				return err
			}
		}
		return nil
	}).Error
}

func listSQLLogChannelIDs(ctx context.Context, db *gorm.DB, filter logFilter) (map[int]struct{}, error) {
	filter.NonzeroChannel = true
	tx, err := filter.sqlQuery(db.WithContext(ctx))
	if err != nil {
		return nil, err
	}
	var values []int
	if err := tx.Distinct("channel_id").Pluck("channel_id", &values).Error; err != nil {
		return nil, err
	}
	ids := make(map[int]struct{}, len(values))
	for _, id := range values {
		ids[id] = struct{}{}
	}
	return ids, nil
}
