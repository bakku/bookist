package testsupport

import (
	"context"
	"database/sql"
	"testing"
)

type ReadRow struct {
	ID         string
	BookID     string
	StartedAt  *string
	FinishedAt *string
	Rating     *float64
	Notes      *string
	CreatedAt  string
}

func InsertReadRow(t testing.TB, db *sql.DB, row ReadRow) {
	t.Helper()

	_, err := db.ExecContext(context.Background(), `
		INSERT INTO reads (id, book_id, started_at, finished_at, rating, notes, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
	`, row.ID, row.BookID, row.StartedAt, row.FinishedAt, row.Rating, row.Notes, row.CreatedAt, row.CreatedAt)
	if err != nil {
		t.Fatal(err)
	}
}
