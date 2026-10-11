package reads_test

import (
	"context"
	"errors"
	"testing"

	"bakku.dev/bookist/internal/reads"
	"bakku.dev/bookist/internal/testsupport"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// ── Create Constraints ─────────────────────────────────────────────────────────

func TestCreateCheckConstraints(t *testing.T) {
	for _, test := range []struct {
		name  string
		input reads.CreateReadRequest
		want  error
	}{
		{name: "started date", input: reads.CreateReadRequest{StartedAt: new("invalid")}, want: reads.ErrInvalidStartedAt},
		{name: "finished date", input: reads.CreateReadRequest{FinishedAt: new("invalid")}, want: reads.ErrInvalidFinishedAt},
		{name: "abandoned date", input: reads.CreateReadRequest{AbandonedAt: new("invalid")}, want: reads.ErrInvalidAbandonedAt},
		{name: "rating", input: reads.CreateReadRequest{Rating: new(4.2)}, want: reads.ErrInvalidRating},
		{name: "terminal dates", input: reads.CreateReadRequest{FinishedAt: new("2026-02-02"), AbandonedAt: new("2026-02-03")}, want: reads.ErrConflictingTerminalDates},
		{name: "finished order", input: reads.CreateReadRequest{StartedAt: new("2026-02-02"), FinishedAt: new("2026-02-01")}, want: reads.ErrFinishedBeforeStarted},
		{name: "abandoned order", input: reads.CreateReadRequest{StartedAt: new("2026-02-02"), AbandonedAt: new("2026-02-01")}, want: reads.ErrAbandonedBeforeStarted},
	} {
		t.Run(test.name, func(t *testing.T) {
			db := testsupport.OpenMigratedDB(t)
			bookID := testsupport.InsertBookRow(t, db, "Dune", nil)

			_, err := reads.NewSQLiteRepository(db).Create(context.Background(), bookID, test.input)
			if !errors.Is(err, test.want) {
				t.Fatalf("expected %v, got %v", test.want, err)
			}

			testsupport.AssertSQLCount(t, db, 0, `SELECT count(*) FROM reads`)
		})
	}
}

func TestReadConstraintLookalike(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)
	if _, err := db.Exec(`CREATE TRIGGER blocked BEFORE INSERT ON reads BEGIN SELECT RAISE(ABORT, 'FOREIGN KEY constraint failed'); END`); err != nil {
		t.Fatal(err)
	}

	_, err := reads.NewSQLiteRepository(db).Create(context.Background(), bookID, reads.CreateReadRequest{})
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) || sqliteErr.Code() != sqlite3.SQLITE_CONSTRAINT_TRIGGER {
		t.Fatalf("expected unchanged trigger error, got %v", err)
	}
}
