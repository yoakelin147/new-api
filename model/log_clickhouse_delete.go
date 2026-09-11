package model

import (
	"context"
	"fmt"
	"time"
)

func (s *clickHouseLogStore) DeleteBefore(ctx context.Context, timestamp int64) (int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Minute)
	defer cancel()
	// Delete the IDs actually selected, rather than claiming a pre-count of an
	// open-ended timestamp predicate is the number deleted under concurrent writes.
	var upper int64
	if err := s.db.QueryRowContext(ctx, "SELECT coalesce(max(id), 0) FROM "+s.table()+" WHERE created_at < ?", timestamp).Scan(&upper); err != nil {
		return 0, err
	}
	var total, cursor int64
	for {
		rows, err := s.db.QueryContext(ctx, "SELECT id FROM "+s.table()+" WHERE created_at < ? AND id > ? AND id <= ? ORDER BY id LIMIT 1000", timestamp, cursor, upper)
		if err != nil {
			return total, err
		}
		ids := []int64{}
		for rows.Next() {
			var id int64
			if err = rows.Scan(&id); err != nil {
				rows.Close()
				return total, err
			}
			ids = append(ids, id)
		}
		err = rows.Err()
		rows.Close()
		if err != nil {
			return total, err
		}
		if len(ids) == 0 {
			return total, nil
		}
		table := s.table()
		if _, err := s.db.ExecContext(ctx, "ALTER TABLE "+table+" DELETE WHERE id IN (?) SETTINGS mutations_sync = 2", ids); err != nil {
			return total, err
		}
		var remaining int64
		if err := s.db.QueryRowContext(ctx, "SELECT count() FROM "+s.table()+" WHERE id IN (?)", ids).Scan(&remaining); err != nil {
			return total, err
		}
		if remaining != 0 {
			return total, fmt.Errorf("ClickHouse log deletion not yet visible (%d records remain)", remaining)
		}
		total += int64(len(ids))
		cursor = ids[len(ids)-1]
	}
}
