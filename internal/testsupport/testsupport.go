package testsupport

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"bakku.dev/bookist/internal/books"
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

func NewBookService(t testing.TB) *books.Service {
	t.Helper()

	return books.NewService(books.NewSQLiteRepository(OpenMigratedDB(t)))
}

func InsertBookRow(t testing.TB, db *sql.DB, title string, isbn *string) int {
	t.Helper()

	now := "2026-01-02T03:04:05Z"
	isbnValue := sql.NullString{}
	if isbn != nil {
		isbnValue = sql.NullString{String: *isbn, Valid: true}
	}

	var id int
	err := db.QueryRowContext(context.Background(), `
		INSERT INTO books (title, isbn, created_at, updated_at)
		VALUES (?, ?, ?, ?)
		RETURNING id
	`, title, isbnValue, now, now).Scan(&id)
	if err != nil {
		t.Fatal(err)
	}

	return id
}

func AssertBookRow(t testing.TB, db *sql.DB, id int, wantTitle string, wantISBN *string) {
	t.Helper()

	var title string
	var isbn sql.NullString
	var createdAt string
	var updatedAt string
	err := db.QueryRowContext(context.Background(), `
		SELECT title, isbn, created_at, updated_at
		FROM books
		WHERE id = ?
	`, id).Scan(&title, &isbn, &createdAt, &updatedAt)
	if err != nil {
		t.Fatal(err)
	}

	if title != wantTitle {
		t.Fatalf("expected persisted title %q, got %q", wantTitle, title)
	}
	if wantISBN == nil && isbn.Valid {
		t.Fatalf("expected persisted ISBN to be NULL, got %q", isbn.String)
	}
	if wantISBN != nil && (!isbn.Valid || isbn.String != *wantISBN) {
		t.Fatalf("expected persisted ISBN %q, got %#v", *wantISBN, isbn)
	}
	if _, err := time.Parse(time.RFC3339, createdAt); err != nil {
		t.Fatalf("expected RFC3339 created_at, got %q", createdAt)
	}
	if _, err := time.Parse(time.RFC3339, updatedAt); err != nil {
		t.Fatalf("expected RFC3339 updated_at, got %q", updatedAt)
	}
}

func AssertBookCount(t testing.TB, db *sql.DB, want int) {
	t.Helper()

	var count int
	if err := db.QueryRowContext(context.Background(), `SELECT COUNT(*) FROM books`).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != want {
		t.Fatalf("expected %d persisted books, got %d", want, count)
	}
}

func StringPtr(value string) *string {
	return &value
}
