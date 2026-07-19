package testsupport

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/books"
)

func NewBookService(t testing.TB) (*books.Service, *sql.DB) {
	t.Helper()

	db := OpenMigratedDB(t)
	authorRepo := authors.NewSQLiteRepository(db)
	return books.NewService(books.NewSQLiteRepository(db), authorRepo), db
}

func InsertBookRow(t testing.TB, db *sql.DB, title string, isbn *string) int64 {
	t.Helper()

	now := "2026-01-02T03:04:05Z"
	isbnValue := sql.NullString{}
	if isbn != nil {
		isbnValue = sql.NullString{String: *isbn, Valid: true}
	}

	var id int64
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

type BookRowAssertion struct {
	Title             string
	ISBN              *string
	Language          *string
	Publisher         *string
	Edition           *string
	Format            *string
	PurchasedAt       *string
	Pages             *int
	Notes             *string
	Summary           *string
	SeriesName        *string
	SeriesPosition    *float64
	Location          *string
	Condition         *string
	AcquisitionSource *string
	PublishedYear     *int
	PublishedMonth    *int
	PublishedDay      *int
}

func AssertBookRow(t testing.TB, db *sql.DB, id int64, wantTitle string, wantISBN *string) {
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

func AssertBookRowFields(t testing.TB, db *sql.DB, id int64, want BookRowAssertion) {
	t.Helper()

	var title string
	var isbn sql.NullString
	var language sql.NullString
	var publisher sql.NullString
	var edition sql.NullString
	var format sql.NullString
	var purchasedAt sql.NullString
	var notes sql.NullString
	var summary sql.NullString
	var seriesName sql.NullString
	var seriesPosition sql.NullFloat64
	var location sql.NullString
	var condition sql.NullString
	var acquisitionSource sql.NullString
	var pages sql.NullInt64
	var publishedYear sql.NullInt64
	var publishedMonth sql.NullInt64
	var publishedDay sql.NullInt64
	var createdAt string
	var updatedAt string

	err := db.QueryRowContext(context.Background(), `
		SELECT title, isbn, language, publisher, edition, format, purchased_at,
		    pages, notes, summary, series_name, series_position, location,
		    condition, acquisition_source, published_year, published_month, published_day,
		    created_at, updated_at
		FROM books
		WHERE id = ?
	`, id).Scan(&title, &isbn, &language, &publisher, &edition,
		&format, &purchasedAt, &pages, &notes, &summary, &seriesName, &seriesPosition,
		&location, &condition, &acquisitionSource, &publishedYear, &publishedMonth,
		&publishedDay, &createdAt, &updatedAt)
	if err != nil {
		t.Fatal(err)
	}

	if title != want.Title {
		t.Fatalf("expected title %q, got %q", want.Title, title)
	}

	assertNullString(t, "isbn", isbn, want.ISBN)
	assertNullString(t, "language", language, want.Language)
	assertNullString(t, "publisher", publisher, want.Publisher)
	assertNullString(t, "edition", edition, want.Edition)
	assertNullString(t, "format", format, want.Format)
	assertNullString(t, "purchased_at", purchasedAt, want.PurchasedAt)
	assertNullInt(t, "pages", pages, want.Pages)
	assertNullString(t, "notes", notes, want.Notes)
	assertNullString(t, "summary", summary, want.Summary)
	assertNullString(t, "series_name", seriesName, want.SeriesName)
	assertNullFloat(t, "series_position", seriesPosition, want.SeriesPosition)
	assertNullString(t, "location", location, want.Location)
	assertNullString(t, "condition", condition, want.Condition)
	assertNullString(t, "acquisition_source", acquisitionSource, want.AcquisitionSource)
	assertNullInt(t, "published_year", publishedYear, want.PublishedYear)
	assertNullInt(t, "published_month", publishedMonth, want.PublishedMonth)
	assertNullInt(t, "published_day", publishedDay, want.PublishedDay)

	if _, err := time.Parse(time.RFC3339, createdAt); err != nil {
		t.Fatalf("expected RFC3339 created_at, got %q", createdAt)
	}

	if _, err := time.Parse(time.RFC3339, updatedAt); err != nil {
		t.Fatalf("expected RFC3339 updated_at, got %q", updatedAt)
	}
}

func assertNullFloat(t testing.TB, name string, got sql.NullFloat64, want *float64) {
	t.Helper()

	if want == nil && got.Valid {
		t.Fatalf("expected %s to be NULL, got %v", name, got.Float64)
	}

	if want != nil && (!got.Valid || got.Float64 != *want) {
		t.Fatalf("expected %s %v, got %#v", name, *want, got)
	}
}

func assertNullString(t testing.TB, name string, got sql.NullString, want *string) {
	t.Helper()

	if want == nil && got.Valid {
		t.Fatalf("expected %s to be NULL, got %q", name, got.String)
	}

	if want != nil && (!got.Valid || got.String != *want) {
		t.Fatalf("expected %s %q, got %#v", name, *want, got)
	}
}

func assertNullInt(t testing.TB, name string, got sql.NullInt64, want *int) {
	t.Helper()

	if want == nil && got.Valid {
		t.Fatalf("expected %s to be NULL, got %d", name, got.Int64)
	}

	if want != nil && (!got.Valid || int(got.Int64) != *want) {
		t.Fatalf("expected %s %d, got %#v", name, *want, got)
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

func InsertBookAuthorRow(t testing.TB, db *sql.DB, bookID int64, authorID int64) {
	t.Helper()
	now := "2026-01-02T03:04:05Z"

	_, err := db.ExecContext(context.Background(), `
		INSERT INTO book_authors (book_id, author_id, created_at, updated_at)
		VALUES (?, ?, ?, ?)
	`, bookID, authorID, now, now)
	if err != nil {
		t.Fatal(err)
	}
}

func AssertBookAuthors(t testing.TB, db *sql.DB, bookID int64, wantAuthorIDs ...int64) {
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

	var gotIDs []int64
	for rows.Next() {
		var id int64
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
	gotSet := make(map[int64]bool, len(gotIDs))
	for _, id := range gotIDs {
		if gotSet[id] {
			t.Fatalf("duplicate author_id %d in book_authors", id)
		}
		gotSet[id] = true
	}
	for _, want := range wantAuthorIDs {
		if !gotSet[want] {
			t.Fatalf("expected author_id %d not found in book_authors", want)
		}
	}
}
