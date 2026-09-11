package model

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"net"
	"net/url"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/ClickHouse/clickhouse-go/v2"
	"github.com/QuantumNous/new-api/common"
)

// Set once during InitLogDB, before serving requests. A nil store preserves the
// existing LOG_DB implementation and its independently configured SQL dialect.
var clickHouseLogs *clickHouseLogStore

type clickHouseLogStore struct {
	db       *sql.DB
	database string
	ids      *logIDGenerator
}

var clickHouseIdentifier = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)

func quoteClickHouseIdentifier(s string) string { return "`" + s + "`" }

func (s *clickHouseLogStore) table() string {
	return quoteClickHouseIdentifier(s.database) + ".`logs`"
}

func newClickHouseLogStore(ctx context.Context, dsn string, node int) (*clickHouseLogStore, error) {
	// Validate without echoing the DSN: url.Parse errors can contain passwords.
	u, err := url.Parse(strings.TrimSpace(dsn))
	if err != nil || u == nil || (u.Scheme != "clickhouse" && u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
		return nil, errors.New("invalid CLICKHOUSE_DSN (expected clickhouse://, http:// or https://)")
	}
	for _, address := range strings.Split(u.Host, ",") {
		_, port, err := net.SplitHostPort(address)
		if err != nil {
			return nil, errors.New("CLICKHOUSE_DSN must specify a valid host and port")
		}
		if n, err := strconv.Atoi(port); err != nil || n < 1 || n > 65535 {
			return nil, errors.New("invalid CLICKHOUSE_DSN port")
		}
	}
	options, err := clickhouse.ParseDSN(strings.TrimSpace(dsn))
	if err != nil {
		return nil, errors.New("invalid CLICKHOUSE_DSN options")
	}
	database := options.Auth.Database
	if u.Path == "" || u.Path == "/" {
		database = "logs"
	}
	if !clickHouseIdentifier.MatchString(database) {
		return nil, errors.New("ClickHouse database names must contain only letters, digits and underscores")
	}
	if node < 0 || node > 1023 {
		return nil, errors.New("CLICKHOUSE_NODE_ID must be between 0 and 1023")
	}
	if strconv.IntSize != 64 {
		return nil, errors.New("ClickHouse logs require a 64-bit New API build")
	}
	if options.Settings == nil {
		options.Settings = clickhouse.Settings{}
	}
	// Wait for server-side insert batching to finish before reporting success.
	if _, configured := options.Settings["async_insert"]; !configured {
		options.Settings["async_insert"] = 1
		options.Settings["async_insert_busy_timeout_ms"] = 100
	}
	options.Settings["wait_for_async_insert"] = 1
	options.Debug = false
	if options.DialTimeout == 0 {
		options.DialTimeout = 10 * time.Second
	}
	// Connect to an existing database first, so the logs database can be created.
	options.Auth.Database = "default"
	db := clickhouse.OpenDB(options)
	s := &clickHouseLogStore{db: db, database: database}
	ctx, cancel := context.WithTimeout(ctx, 2*time.Minute)
	defer cancel()
	if err := db.PingContext(ctx); err != nil {
		db.Close()
		return nil, fmt.Errorf("connect ClickHouse logs: %w", err)
	}
	if err := s.initialize(ctx); err != nil {
		db.Close()
		return nil, err
	}
	var maxID int64
	if err := db.QueryRowContext(ctx, "SELECT coalesce(max(id), 0) FROM "+s.table()).Scan(&maxID); err != nil {
		db.Close()
		return nil, fmt.Errorf("read ClickHouse log ID maximum: %w", err)
	}
	s.ids, err = newLogIDGenerator(node, maxID, time.Now)
	if err != nil {
		db.Close()
		return nil, err
	}
	return s, nil
}

func initClickHouseLogs(dsn string) error {
	node := 0
	if raw := os.Getenv("CLICKHOUSE_NODE_ID"); raw != "" {
		var err error
		node, err = strconv.Atoi(raw)
		if err != nil {
			return errors.New("invalid CLICKHOUSE_NODE_ID")
		}
	}
	s, err := newClickHouseLogStore(context.Background(), dsn, node)
	if err != nil {
		return err
	}
	clickHouseLogs = s
	common.UsingClickHouse = true
	common.SysLog("using ClickHouse for logs")
	return nil
}

func (s *clickHouseLogStore) Close() error { return s.db.Close() }

func createLog(ctx context.Context, log *Log) error {
	if clickHouseLogs != nil {
		return clickHouseLogs.Create(ctx, log)
	}
	return LOG_DB.Create(log).Error
}
