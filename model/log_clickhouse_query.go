package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"
)

type logOrder string

const (
	logOrderID      logOrder = "id DESC"
	logOrderCreated logOrder = "created_at DESC, id DESC"
)

type logFilter struct {
	Types             []int
	UserIDs           []int // nil is unfiltered; empty matches no users.
	TokenID           *int
	Start, End        int64
	Username          string
	TokenName         string
	ModelName         string
	ModelContains     bool
	ExactText         bool
	ChannelID         int
	Group             string
	RequestID         string
	UpstreamRequestID string
	StreamStatus      string
	NonzeroChannel    bool
	NonzeroUser       bool
	HasToken          bool
}

func clickHouseLike(value string, contains bool) (string, error) {
	if !contains {
		if _, err := sanitizeLikePattern(value); err != nil {
			return "", err
		}
	}
	var pattern strings.Builder
	if contains {
		pattern.WriteByte('%')
	}
	for _, c := range value {
		if c == '\\' || c == '_' || (contains && c == '%') {
			pattern.WriteByte('\\')
		}
		pattern.WriteRune(c)
	}
	if contains {
		pattern.WriteByte('%')
	}
	return pattern.String(), nil
}

func (f logFilter) where() (string, []any, error) {
	clauses := []string{"1 = 1"}
	args := []any{}
	add := func(clause string, value any) { clauses = append(clauses, clause); args = append(args, value) }
	if f.Types != nil {
		if len(f.Types) == 0 {
			clauses = append(clauses, "0 = 1")
		} else {
			add("type IN (?)", f.Types)
		}
	}
	if f.UserIDs != nil {
		if len(f.UserIDs) == 0 {
			clauses = append(clauses, "0 = 1")
		} else {
			add("user_id IN (?)", f.UserIDs)
		}
	}
	if f.TokenID != nil {
		add("token_id = ?", *f.TokenID)
	}
	if f.Start != 0 {
		add("created_at >= ?", f.Start)
	}
	if f.End != 0 {
		add("created_at <= ?", f.End)
	}
	for _, text := range []struct {
		column, value string
		contains      bool
	}{{"username", f.Username, false}, {"model_name", f.ModelName, f.ModelContains}} {
		if text.value == "" {
			continue
		}
		if !f.ExactText && (text.contains || strings.Contains(text.value, "%")) {
			pattern, err := clickHouseLike(text.value, text.contains)
			if err != nil {
				return "", nil, err
			}
			add(text.column+" LIKE ?", pattern)
		} else {
			add(text.column+" = ?", text.value)
		}
	}
	for _, text := range []struct{ column, value string }{{"token_name", f.TokenName}, {"`group`", f.Group}, {"request_id", f.RequestID}, {"upstream_request_id", f.UpstreamRequestID}} {
		if text.value != "" {
			add(text.column+" = ?", text.value)
		}
	}
	if f.ChannelID != 0 {
		add("channel_id = ?", f.ChannelID)
	}
	if f.NonzeroChannel {
		clauses = append(clauses, "channel_id <> 0")
	}
	if f.NonzeroUser {
		clauses = append(clauses, "user_id <> 0")
	}
	if f.HasToken {
		clauses = append(clauses, "(token_id <> 0 OR token_name <> '')")
	}
	if f.StreamStatus != "" {
		switch f.StreamStatus {
		case StreamStatusFilterError, StreamStatusFilterClientGone, StreamStatusFilterTimeout, StreamStatusFilterScannerError, StreamStatusFilterHandlerStop, StreamStatusFilterPanic, StreamStatusFilterPingFail:
		default:
			return "", nil, fmt.Errorf("invalid stream status filter: %s", f.StreamStatus)
		}
		// Preserve the existing raw-text filter, including historical JSON spacing.
		add("position(other, ?) > 0", `"stream_status":{`)
		add("position(other, ?) > 0", `"status":"error"`)
		if f.StreamStatus != StreamStatusFilterError {
			add("position(other, ?) > 0", `"end_reason":"`+f.StreamStatus+`"`)
		}
	}
	return strings.Join(clauses, " AND "), args, nil
}

func (s *clickHouseLogStore) Create(ctx context.Context, log *Log) error {
	if s.failover != nil {
		return s.failover.Write(ctx, log)
	}
	return s.createDirect(ctx, log)
}

func (s *clickHouseLogStore) createDirect(ctx context.Context, log *Log) error {
	ctx, cancel := context.WithTimeout(ctx, time.Minute)
	defer cancel()
	if err := s.prepare(ctx); err != nil {
		return err
	}
	if log.Id == 0 {
		id, err := s.ids.Next(ctx)
		if err != nil {
			return err
		}
		log.Id = int(id)
	}
	if log.Id < 0 || int64(log.Id) > maxLogID {
		return errors.New("invalid ClickHouse log ID")
	}
	columns := []string{}
	for _, col := range clickHouseLogColumns {
		if col.name != "channel_name" {
			columns = append(columns, quoteClickHouseIdentifier(col.name))
		}
	}
	_, err := s.db.ExecContext(ctx, "INSERT INTO "+s.table()+" ("+strings.Join(columns, ",")+") VALUES ("+strings.TrimSuffix(strings.Repeat("?,", len(columns)), ",")+")",
		log.Id, log.UserId, log.CreatedAt, log.Type, log.Content, log.Username, log.TokenName, log.ModelName,
		log.Quota, log.PromptTokens, log.CompletionTokens, log.UseTime, log.IsStream, log.ChannelId, log.TokenId,
		log.Group, log.Ip, log.RequestId, log.UpstreamRequestId, log.Other)
	return err
}

func (s *clickHouseLogStore) scanQuery(ctx context.Context, suffix string, args []any, visit func(Log) error) error {
	columns := []string{}
	for _, col := range clickHouseLogColumns {
		fallback := "0"
		if col.kind == "String" {
			fallback = "''"
		}
		columns = append(columns, "coalesce("+quoteClickHouseIdentifier(col.name)+", "+fallback+")")
	}
	rows, err := s.db.QueryContext(ctx, "SELECT "+strings.Join(columns, ",")+" FROM "+s.table()+" "+suffix, args...)
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var log Log
		if err := scanClickHouseLog(rows, &log); err != nil {
			return err
		}
		if err := visit(log); err != nil {
			return err
		}
	}
	return rows.Err()
}

func scanClickHouseLog(rows *sql.Rows, log *Log) error {
	return rows.Scan(&log.Id, &log.UserId, &log.CreatedAt, &log.Type, &log.Content, &log.Username, &log.TokenName, &log.ModelName,
		&log.Quota, &log.PromptTokens, &log.CompletionTokens, &log.UseTime, &log.IsStream, &log.ChannelId, &log.ChannelName,
		&log.TokenId, &log.Group, &log.Ip, &log.RequestId, &log.UpstreamRequestId, &log.Other)
}

func (s *clickHouseLogStore) Query(ctx context.Context, filter logFilter, offset, limit int, order logOrder) ([]*Log, int64, error) {
	if offset < 0 || limit <= 0 {
		return nil, 0, errors.New("invalid log pagination")
	}
	if order != logOrderID && order != logOrderCreated {
		return nil, 0, errors.New("invalid log order")
	}
	where, args, err := filter.where()
	if err != nil {
		return nil, 0, err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	var total int64
	if err := s.db.QueryRowContext(ctx, "SELECT count() FROM "+s.table()+" WHERE "+where, args...).Scan(&total); err != nil {
		return nil, 0, err
	}
	logs := make([]*Log, 0)
	err = s.scanQuery(ctx, "WHERE "+where+" ORDER BY "+string(order)+" LIMIT ? OFFSET ?", append(args, limit, offset), func(log Log) error { logs = append(logs, &log); return nil })
	return logs, total, err
}

func (s *clickHouseLogStore) Scan(ctx context.Context, filter logFilter, visit func(Log) error) error {
	where, args, err := filter.where()
	if err != nil {
		return err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	return s.scanQuery(ctx, "WHERE "+where+" ORDER BY id", args, visit)
}

func (s *clickHouseLogStore) ChannelIDs(ctx context.Context, filter logFilter) (map[int]struct{}, error) {
	where, args, err := filter.where()
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	rows, err := s.db.QueryContext(ctx, "SELECT channel_id FROM "+s.table()+" WHERE "+where+" AND channel_id <> 0", args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	ids := map[int]struct{}{}
	for rows.Next() {
		var id int
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids[id] = struct{}{}
	}
	return ids, rows.Err()
}
