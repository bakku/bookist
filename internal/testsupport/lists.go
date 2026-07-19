package testsupport

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
)

func InsertListRow(t testing.TB, db *sql.DB, name string) string {
	t.Helper()

	now := "2026-01-02T03:04:05Z"
	id := uuid.NewString()
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO lists (id, name, created_at, updated_at)
		VALUES (?, ?, ?, ?)
	`, id, name, now, now)
	if err != nil {
		t.Fatal(err)
	}

	return id
}

func InsertListRowWithDescription(t testing.TB, db *sql.DB, name string, description string) string {
	t.Helper()

	now := "2026-01-02T03:04:05Z"
	id := uuid.NewString()
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO lists (id, name, description, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, id, name, description, now, now)
	if err != nil {
		t.Fatal(err)
	}

	return id
}

func AssertListRow(t testing.TB, db *sql.DB, id string, wantName string) {
	t.Helper()

	var name string
	var createdAt string
	var updatedAt string
	err := db.QueryRowContext(context.Background(), `
		SELECT name, created_at, updated_at
		FROM lists
		WHERE id = ?
	`, id).Scan(&name, &createdAt, &updatedAt)
	if err != nil {
		t.Fatal(err)
	}

	if name != wantName {
		t.Fatalf("expected persisted name %q, got %q", wantName, name)
	}

	if _, err := time.Parse(time.RFC3339, createdAt); err != nil {
		t.Fatalf("expected RFC3339 created_at, got %q", createdAt)
	}

	if _, err := time.Parse(time.RFC3339, updatedAt); err != nil {
		t.Fatalf("expected RFC3339 updated_at, got %q", updatedAt)
	}
}

func AssertListCount(t testing.TB, db *sql.DB, want int) {
	t.Helper()

	var count int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM lists`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("expected %d persisted lists, got %d", want, count)
	}
}

func InsertBookListRow(t testing.TB, db *sql.DB, listID string, bookID string) {
	t.Helper()

	_, err := db.ExecContext(context.Background(), `
		INSERT INTO book_lists (list_id, book_id)
		VALUES (?, ?)
	`, listID, bookID)
	if err != nil {
		t.Fatal(err)
	}
}

func AssertBookListRow(t testing.TB, db *sql.DB, listID string, bookID string) {
	t.Helper()

	var count int
	err := db.QueryRowContext(context.Background(), `
		SELECT COUNT(*) FROM book_lists WHERE list_id = ? AND book_id = ?
	`, listID, bookID).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected 1 book_lists row for list %s book %s, got %d", listID, bookID, count)
	}
}

func AssertBookListCount(t testing.TB, db *sql.DB, listID string, want int) {
	t.Helper()

	var count int
	err := db.QueryRowContext(context.Background(), `
		SELECT COUNT(*) FROM book_lists WHERE list_id = ?
	`, listID).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("expected %d book_lists rows for list %s, got %d", want, listID, count)
	}
}
