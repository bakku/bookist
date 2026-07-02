package testsupport

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"
	"time"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/database"
	"github.com/google/uuid"
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

func NewBookService(t testing.TB) (*books.Service, *sql.DB) {
	t.Helper()

	db := OpenMigratedDB(t)
	authorRepo := authors.NewSQLiteRepository(db)
	return books.NewService(books.NewSQLiteRepository(db), authorRepo), db
}

func InsertBookRow(t testing.TB, db *sql.DB, title string, isbn *string) string {
	t.Helper()

	now := "2026-01-02T03:04:05Z"
	isbnValue := sql.NullString{}
	if isbn != nil {
		isbnValue = sql.NullString{String: *isbn, Valid: true}
	}

	id := uuid.NewString()
	_, err := db.ExecContext(context.Background(), `
		INSERT INTO books (id, title, isbn, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?)
	`, id, title, isbnValue, now, now)
	if err != nil {
		t.Fatal(err)
	}

	return id
}

func AssertBookRow(t testing.TB, db *sql.DB, id string, wantTitle string, wantISBN *string) {
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

func InsertBookAuthorRow(t testing.TB, db *sql.DB, bookID string, authorID string) {
	t.Helper()

	_, err := db.ExecContext(context.Background(), `
		INSERT INTO book_authors (book_id, author_id)
		VALUES (?, ?)
	`, bookID, authorID)
	if err != nil {
		t.Fatal(err)
	}
}

func AssertBookAuthors(t testing.TB, db *sql.DB, bookID string, wantAuthorIDs ...string) {
	t.Helper()

	rows, err := db.QueryContext(context.Background(), `
		SELECT author_id
		FROM book_authors
		WHERE book_id = ?
	`, bookID)
	if err != nil {
		t.Fatal(err)
	}
	defer rows.Close()

	var gotIDs []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			t.Fatal(err)
		}
		gotIDs = append(gotIDs, id)
	}
	if err := rows.Err(); err != nil {
		t.Fatal(err)
	}

	if len(gotIDs) != len(wantAuthorIDs) {
		t.Fatalf("expected %d book_authors rows, got %d", len(wantAuthorIDs), len(gotIDs))
	}
	gotSet := make(map[string]bool, len(gotIDs))
	for _, id := range gotIDs {
		if gotSet[id] {
			t.Fatalf("duplicate author_id %q in book_authors", id)
		}
		gotSet[id] = true
	}
	for _, want := range wantAuthorIDs {
		if !gotSet[want] {
			t.Fatalf("expected author_id %q not found in book_authors", want)
		}
	}
}

func StringPtr(value string) *string {
	return &value
}
