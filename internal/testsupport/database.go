package testsupport

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"bakku.dev/bookist/internal/database"
)

func OpenMigratedDB(t testing.TB) *sql.DB {
	t.Helper()

	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "bookist.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := database.MigrateUp(db); err != nil {
		t.Fatal(err)
	}

	return db
}

func AssertSQLCount(t testing.TB, db *sql.DB, want int, query string, args ...any) {
	t.Helper()

	var got int
	if err := db.QueryRowContext(context.Background(), query, args...).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("expected count %d, got %d", want, got)
	}
}
