package testsupport

import (
	"context"
	"database/sql"
	"testing"
	"time"
)

func InsertListRow(t testing.TB, db *sql.DB, name string) int64 {
	t.Helper()

	now := "2026-01-02T03:04:05Z"
	var id int64
	err := db.QueryRowContext(context.Background(), `
		INSERT INTO lists (name, created_at, updated_at)
		VALUES (?, ?, ?)
		RETURNING id
	`, name, now, now).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}

	return id
}

func InsertListRowWithDescription(t testing.TB, db *sql.DB, name string, description string) int64 {
	t.Helper()

	now := "2026-01-02T03:04:05Z"
	var id int64
	err := db.QueryRowContext(context.Background(), `
		INSERT INTO lists (name, description, created_at, updated_at)
		VALUES (?, ?, ?, ?)
		RETURNING id
	`, name, description, now, now).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}

	return id
}

func AssertListRow(t testing.TB, db *sql.DB, id int64, wantName string) {
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

func InsertBookListRow(t testing.TB, db *sql.DB, listID int64, bookID int64) {
	t.Helper()
	now := "2026-01-02T03:04:05Z"

	_, err := db.ExecContext(context.Background(), `
		INSERT INTO book_lists (list_id, book_id, created_at, updated_at)
		VALUES (?, ?, ?, ?)
	`, listID, bookID, now, now)
	if err != nil {
		t.Fatal(err)
	}
}

func AssertBookListRow(t testing.TB, db *sql.DB, listID int64, bookID int64) {
	t.Helper()

	var count int
	err := db.QueryRowContext(context.Background(), `
		SELECT COUNT(*) FROM book_lists WHERE list_id = ? AND book_id = ?
	`, listID, bookID).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatalf("expected 1 book_lists row for list %d book %d, got %d", listID, bookID, count)
	}
}

func AssertBookListCount(t testing.TB, db *sql.DB, listID int64, want int) {
	t.Helper()

	var count int
	err := db.QueryRowContext(context.Background(), `
		SELECT COUNT(*) FROM book_lists WHERE list_id = ?
	`, listID).Scan(&count)
	if err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("expected %d book_lists rows for list %d, got %d", want, listID, count)
	}
}
