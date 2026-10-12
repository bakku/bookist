package testsupport

import (
	"context"
	"database/sql"
	"testing"
)

func InsertListRow(t testing.TB, db *sql.DB, name string) int64 {
	t.Helper()

	now := fixtureTimestamp
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

func InsertListRowWithDescription(t testing.TB, db *sql.DB, name, description string) int64 {
	t.Helper()

	now := fixtureTimestamp
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

func AssertListRow(t testing.TB, db *sql.DB, id int64, wantName string, wantDescription *string) {
	t.Helper()

	var name string
	var description sql.NullString
	var createdAt string
	var updatedAt string

	err := db.QueryRowContext(context.Background(), `
		SELECT name, description, created_at, updated_at
		FROM lists
		WHERE id = ?
	`, id).Scan(&name, &description, &createdAt, &updatedAt)
	if err != nil {
		t.Fatal(err)
	}

	if name != wantName {
		t.Fatalf("expected persisted name %q, got %q", wantName, name)
	}

	if wantDescription == nil && description.Valid {
		t.Fatalf("expected description to be nil, got %q", description.String)
	}

	if wantDescription != nil && (!description.Valid || description.String != *wantDescription) {
		t.Fatalf("expected persisted description %q, got %q", *wantDescription, description.String)
	}

	assertTimestamp(t, "created_at", createdAt)

	assertTimestamp(t, "updated_at", updatedAt)
}

func InsertBookListRow(t testing.TB, db *sql.DB, listID int64, bookID int64) {
	t.Helper()
	now := fixtureTimestamp

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
