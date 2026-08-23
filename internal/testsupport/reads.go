package testsupport

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

type ReadRow struct {
	ID          int64
	BookID      int64
	StartedAt   *string
	FinishedAt  *string
	AbandonedAt *string
	Rating      *float64
	Notes       *string
	CreatedAt   string
}

type ReadRowAssertion struct {
	BookID      int64
	StartedAt   *string
	FinishedAt  *string
	AbandonedAt *string
	Rating      *float64
	Notes       *string
}

func InsertReadRow(t testing.TB, db *sql.DB, row ReadRow) {
	t.Helper()

	_, err := db.ExecContext(context.Background(), `
		INSERT INTO reads (id, book_id, started_at, finished_at, abandoned_at, rating, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, row.ID, row.BookID, row.StartedAt, row.FinishedAt, row.AbandonedAt, row.Rating, row.Notes, row.CreatedAt, row.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
}

func AssertReadRow(t testing.TB, db *sql.DB, id int64, want ReadRowAssertion) {
	t.Helper()

	var bookID int64
	var startedAt, finishedAt, abandonedAt, notes sql.NullString
	var rating sql.NullFloat64
	var createdAt, updatedAt string

	err := db.QueryRowContext(context.Background(), `
		SELECT book_id, started_at, finished_at, abandoned_at, rating, notes, created_at, updated_at
		FROM reads
		WHERE id = ?
	`, id).Scan(&bookID, &startedAt, &finishedAt, &abandonedAt, &rating, &notes, &createdAt, &updatedAt)
	if err != nil {
		t.Fatal(err)
	}

	if bookID != want.BookID {
		t.Fatalf("expected book_id %d, got %d", want.BookID, bookID)
	}

	assertNullString(t, "started_at", startedAt, want.StartedAt)
	assertNullString(t, "finished_at", finishedAt, want.FinishedAt)
	assertNullString(t, "abandoned_at", abandonedAt, want.AbandonedAt)
	assertNullFloat(t, "rating", rating, want.Rating)
	assertNullString(t, "notes", notes, want.Notes)

	if _, err := time.Parse(time.RFC3339, createdAt); err != nil {
		t.Fatalf("expected RFC3339 created_at, got %q", createdAt)
	}

	if _, err := time.Parse(time.RFC3339, updatedAt); err != nil {
		t.Fatalf("expected RFC3339 updated_at, got %q", updatedAt)
	}
}
