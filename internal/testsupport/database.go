package testsupport

import (
	"context"
	"database/sql"
	"log/slog"
	"path/filepath"
	"testing"
	"time"

	"bakku.dev/bookist/internal/database"
)

func OpenMigratedDB(t testing.TB) *sql.DB {
	t.Helper()

	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "bookist.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := database.MigrateUp(context.Background(), db, slog.New(slog.DiscardHandler)); err != nil {
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

// fixtureTimestamp is the created_at and updated_at of seeded rows. It uses the
// stored layout so fixtures sort consistently with rows written by repositories.
var fixtureTimestamp = database.FormatTime(time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC))

// assertTimestamp verifies value is in the layout every writer uses.
func assertTimestamp(t testing.TB, column, value string) {
	t.Helper()

	if _, err := time.Parse(database.TimeLayout, value); err != nil {
		t.Fatalf("expected %s in layout %q, got %q", column, database.TimeLayout, value)
	}
}
