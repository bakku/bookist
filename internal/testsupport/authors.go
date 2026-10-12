package testsupport

import (
	"context"
	"database/sql"
	"testing"
)

func InsertAuthorRow(t testing.TB, db *sql.DB, name string) int64 {
	t.Helper()

	now := fixtureTimestamp
	var id int64
	err := db.QueryRowContext(context.Background(), `
		INSERT INTO authors (name, created_at, updated_at)
		VALUES (?, ?, ?)
		RETURNING id
	`, name, now, now).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}

	return id
}

func AssertAuthorRow(t testing.TB, db *sql.DB, id int64, wantName string) {
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

	assertTimestamp(t, "created_at", createdAt)

	assertTimestamp(t, "updated_at", updatedAt)
}
