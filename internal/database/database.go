package database

import (
	"context"
	"database/sql"
	"strings"

	_ "modernc.org/sqlite"
)

// connectionParams configure every pooled connection. busy_timeout makes
// writers wait for the lock instead of failing with SQLITE_BUSY, WAL lets reads
// proceed during a write, and immediate transactions take the write lock up
// front so a read-then-write transaction cannot fail while upgrading it.
const connectionParams = "_pragma=busy_timeout(5000)" +
	"&_pragma=journal_mode(WAL)" +
	"&_pragma=foreign_keys(1)" +
	"&_txlock=immediate"

func Open(ctx context.Context, path string) (*sql.DB, error) {
	registerFoldFunction()

	separator := "?"
	if strings.Contains(path, "?") {
		separator = "&"
	}

	db, err := sql.Open("sqlite", path+separator+connectionParams)
	if err != nil {
		return nil, err
	}

	if err := db.PingContext(ctx); err != nil {
		_ = db.Close()
		return nil, err
	}

	return db, nil
}
