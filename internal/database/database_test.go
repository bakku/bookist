package database_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"bakku.dev/bookist/internal/database"
)

func TestOpenEnablesForeignKeysOnEveryConnection(t *testing.T) {
	ctx := context.Background()

	db, err := database.Open(ctx, filepath.Join(t.TempDir(), "bookist.db"))
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	db.SetMaxOpenConns(2)

	first := openConnection(t, ctx, db)
	defer func() {
		_ = first.Close()
	}()

	second := openConnection(t, ctx, db)
	defer func() {
		_ = second.Close()
	}()

	assertForeignKeysEnabled(t, ctx, first)
	assertForeignKeysEnabled(t, ctx, second)
}

func openConnection(t *testing.T, ctx context.Context, db *sql.DB) *sql.Conn {
	t.Helper()

	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return conn
}

func assertForeignKeysEnabled(t *testing.T, ctx context.Context, conn *sql.Conn) {
	t.Helper()

	var enabled int
	if err := conn.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&enabled); err != nil {
		t.Fatal(err)
	}
	if enabled != 1 {
		t.Fatalf("expected foreign keys enabled, got %d", enabled)
	}
}
