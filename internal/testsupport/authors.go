package testsupport

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/google/uuid"
)

func InsertAuthorRow(t testing.TB, db *sql.DB, name string) string {
	t.Helper()

	now := "2026-01-02T03:04:05Z"
	id := uuid.NewString()
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO authors (id, name, created_at, updated_at)
		VALUES (?, ?, ?, ?)
	`, id, name, now, now)
	if err != nil {
		t.Fatal(err)
	}

	return id
}

func AssertAuthorRow(t testing.TB, db *sql.DB, id string, wantName string) {
	t.Helper()

	var name string
	var createdAt string
	var updatedAt string
	err := db.QueryRowContext(context.Background(), `
		SELECT name, created_at, updated_at
		FROM authors
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

func AssertAuthorCount(t testing.TB, db *sql.DB, want int) {
	t.Helper()

	var count int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM authors`).Scan(&count); err != nil {
		t.Fatal(err)
	}

	if count != want {
		t.Fatalf("expected %d persisted authors, got %d", want, count)
	}
}
