package model

import (
	"context"
	"fmt"
	"strings"
)

// Keep the existing Log columns. Additional server-side columns are allowed,
// but startup never alters an existing table or rewrites historical records.
var clickHouseLogColumns = []struct{ name, kind string }{
	{"id", "Int64"}, {"user_id", "Int64"}, {"created_at", "Int64"}, {"type", "Int64"},
	{"content", "String"}, {"username", "String"}, {"token_name", "String"}, {"model_name", "String"},
	{"quota", "Int64"}, {"prompt_tokens", "Int64"}, {"completion_tokens", "Int64"}, {"use_time", "Int64"},
	{"is_stream", "Bool"}, {"channel_id", "Int64"}, {"channel_name", "String"}, {"token_id", "Int64"},
	{"group", "String"}, {"ip", "String"}, {"request_id", "String"}, {"upstream_request_id", "String"}, {"other", "String"},
}

func (s *clickHouseLogStore) initialize(ctx context.Context) error {
	var exists uint8
	if err := s.db.QueryRowContext(ctx, "EXISTS DATABASE "+quoteClickHouseIdentifier(s.database)).Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		if _, err := s.db.ExecContext(ctx, "CREATE DATABASE IF NOT EXISTS "+quoteClickHouseIdentifier(s.database)); err != nil {
			return fmt.Errorf("create ClickHouse log database: %w", err)
		}
	}
	if err := s.db.QueryRowContext(ctx, "EXISTS TABLE "+s.table()).Scan(&exists); err != nil {
		return err
	}
	if exists == 0 {
		definitions := make([]string, 0, len(clickHouseLogColumns))
		for _, col := range clickHouseLogColumns {
			value := "0"
			if col.kind == "String" {
				value = "''"
			}
			definitions = append(definitions, quoteClickHouseIdentifier(col.name)+" "+col.kind+" DEFAULT "+value)
		}
		columns := " (" + strings.Join(definitions, ", ") + ")"
		storage := " ENGINE = MergeTree()"
		table := s.table()
		ddl := "CREATE TABLE IF NOT EXISTS " + table + columns + storage +
			" PARTITION BY toYYYYMM(toDateTime(created_at, 'UTC')) ORDER BY (created_at, id)"
		if _, err := s.db.ExecContext(ctx, ddl); err != nil {
			return fmt.Errorf("create ClickHouse logs storage: %w", err)
		}
	}
	var engine string
	if err := s.db.QueryRowContext(ctx, "SELECT engine FROM system.tables WHERE database = ? AND name = 'logs'", s.database).Scan(&engine); err != nil {
		return err
	}
	if !strings.HasSuffix(engine, "MergeTree") {
		return fmt.Errorf("unsupported ClickHouse logs engine %s: this adapter requires a MergeTree log table", engine)
	}
	rows, err := s.db.QueryContext(ctx, "SELECT name, type FROM system.columns WHERE database = ? AND table = 'logs'", s.database)
	if err != nil {
		return err
	}
	defer rows.Close()
	columns := make(map[string]string)
	for rows.Next() {
		var name, kind string
		if err := rows.Scan(&name, &kind); err != nil {
			return err
		}
		columns[name] = kind
	}
	if err := rows.Err(); err != nil {
		return err
	}
	for _, col := range clickHouseLogColumns {
		kind := columns[col.name]
		for strings.HasPrefix(kind, "Nullable(") || strings.HasPrefix(kind, "LowCardinality(") {
			kind = kind[strings.IndexByte(kind, '(')+1 : len(kind)-1]
		}
		valid := kind == col.kind
		if col.kind == "Int64" {
			valid = kind == "Int8" || kind == "Int16" || kind == "Int32" || kind == "Int64" || kind == "UInt8" || kind == "UInt16" || kind == "UInt32" || kind == "UInt64"
		}
		if col.kind == "Bool" {
			valid = kind == "Bool" || kind == "UInt8"
		}
		if col.name == "id" {
			valid = kind == "Int64" || kind == "UInt64"
		}
		if !valid {
			return fmt.Errorf("incompatible ClickHouse logs column %s (%s); existing tables are not altered", col.name, columns[col.name])
		}
	}
	return nil
}
