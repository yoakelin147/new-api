package model

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/QuantumNous/new-api/common"
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
	retryAt := s.readRetryAt.Load()
	attempt := state.source == "clickhouse" || (state.source != "original" && (retryAt == 0 || time.Now().UnixNano() >= retryAt))
	if attempt {
		// Only one reader probes recovery; other requests keep reading the original DB.
		if retryAt != 0 && state.source == "" {
			next := time.Now().Add(logFailoverRetryInterval).UnixNano()
			attempt = s.readRetryAt.CompareAndSwap(retryAt, next)
			retryAt = next
		}
	}
	if attempt {
		if err := primary(ctx); err == nil {
			s.readRetryAt.CompareAndSwap(retryAt, 0)
			state.source = "clickhouse"
			return nil
		} else {
			if s.fallback == nil {
				return err
			}
			previous := s.readRetryAt.Swap(time.Now().Add(logFailoverRetryInterval).UnixNano())
			if previous == 0 {
				common.SysError("ClickHouse log reads degraded to the original database: " + err.Error())
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
