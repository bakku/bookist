package database_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"bakku.dev/bookist/internal/database"
	"bakku.dev/bookist/internal/testsupport"
)

func TestOpenEnablesForeignKeysOnEveryConnection(t *testing.T) {
	ctx := context.Background()

	db, err := database.Open(ctx, filepath.Join(t.TempDir(), "bookist.db"))
	if err != nil {
		t.Fatal(err)
	}

	t.Cleanup(func() {
		_ = db.Close()
	})

	db.SetMaxOpenConns(2)

	first := openConnection(t, ctx, db)
	defer func() {
		_ = first.Close()
	}()

	second := openConnection(t, ctx, db)
	defer func() {
		_ = second.Close()
	}()

	assertForeignKeysEnabled(t, ctx, first)
	assertForeignKeysEnabled(t, ctx, second)
}

func TestInitialSchemaHasIDsAndTimestampsOnEveryTable(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	for _, table := range []string{"books", "authors", "book_authors", "lists", "book_lists", "reads"} {
		t.Run(table, func(t *testing.T) {
			rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
			if err != nil {
				t.Fatal(err)
			}
			defer rows.Close()

			columns := make(map[string]struct {
				kind    string
				notNull bool
				primary bool
			})
			for rows.Next() {
				var cid int
				var name, kind string
				var notNull, primary int
				var defaultValue any
				if err := rows.Scan(&cid, &name, &kind, &notNull, &defaultValue, &primary); err != nil {
					t.Fatal(err)
				}
				columns[name] = struct {
					kind    string
					notNull bool
					primary bool
				}{kind: kind, notNull: notNull == 1, primary: primary == 1}
			}
			if !columns["id"].primary {
				t.Fatal("expected id primary key")
			}
			if columns["id"].kind != "INTEGER" {
				t.Fatalf("expected INTEGER id, got %q", columns["id"].kind)
			}
			if !columns["created_at"].notNull || !columns["updated_at"].notNull {
				t.Fatal("expected non-null created_at and updated_at")
			}
		})
	}
}

func TestInitialSchemaGeneratesIntegerIDs(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	id := testsupport.InsertAuthorRow(t, db, "Jane Austen")
	if id <= 0 {
		t.Fatalf("expected generated positive ID, got %d", id)
	}
}

func TestInitialSchemaEnforcesBookValidation(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	now := "2026-01-02T03:04:05Z"

	for _, test := range []struct {
		title string
		cols  string
		args  []any
	}{
		{title: "invalid pages", cols: "pages", args: []any{0}},
		{title: "month without year", cols: "published_month", args: []any{1}},
		{title: "invalid calendar day", cols: "published_year, published_month, published_day", args: []any{2023, 2, 29}},
		{title: "invalid purchased date", cols: "purchased_at", args: []any{"2025-02-29"}},
		{title: "blank purchase price", cols: "purchase_price", args: []any{" "}},
	} {
		placeholders := "?"
		for range test.args[1:] {
			placeholders += ", ?"
		}
		args := []any{test.title}
		args = append(args, test.args...)
		args = append(args, now, now)
		query := `INSERT INTO books (title, ` + test.cols + `, created_at, updated_at) VALUES (?, ` + placeholders + `, ?, ?)`
		if _, err := db.Exec(query, args...); err == nil {
			t.Fatalf("expected %s to violate a constraint", test.title)
		}
	}
}

func TestInitialSchemaSupportsExtendedBookMetadata(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	now := "2026-01-02T03:04:05Z"

	minimalID := testsupport.InsertBookRow(t, db, "Minimal", nil)
	var summary, seriesName, location, condition, acquisitionSource sql.NullString
	var seriesPosition sql.NullFloat64
	if err := db.QueryRow(`
		SELECT summary, series_name, series_position, location, condition, acquisition_source
		FROM books WHERE id = ?
	`, minimalID).Scan(&summary, &seriesName, &seriesPosition, &location, &condition, &acquisitionSource); err != nil {
		t.Fatal(err)
	}
	if summary.Valid || seriesName.Valid || seriesPosition.Valid || location.Valid || condition.Valid || acquisitionSource.Valid {
		t.Fatal("expected extended metadata to default to NULL")
	}

	for _, value := range []string{"new", "very_good", "good", "acceptable", "poor"} {
		_, err := db.Exec(`
			INSERT INTO books (title, summary, series_name, series_position, location, condition, acquisition_source, created_at, updated_at)
			VALUES (?, 'Summary', 'Series', 1.5, 'Shelf', ?, 'Gift', ?, ?)
		`, "Condition "+value, value, now, now)
		if err != nil {
			t.Fatalf("expected condition %q and fractional position to be accepted: %v", value, err)
		}
	}

	for _, test := range []struct {
		title  string
		column string
		value  any
	}{
		{title: "invalid condition", column: "condition", value: "like_new"},
		{title: "zero series position", column: "series_position", value: 0},
		{title: "negative series position", column: "series_position", value: -1.5},
	} {
		query := `INSERT INTO books (title, ` + test.column + `, created_at, updated_at) VALUES (?, ?, ?, ?)`
		if _, err := db.Exec(query, test.title, test.value, now, now); err == nil {
			t.Fatalf("expected %s to violate a constraint", test.title)
		}
	}
}

func TestInitialSchemaEnforcesRelationshipMetadataAndUniqueness(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)
	authorID := testsupport.InsertAuthorRow(t, db, "Frank Herbert")
	now := "2026-01-02T03:04:05Z"

	if _, err := db.Exec(`INSERT INTO book_authors (book_id, author_id) VALUES (?, ?)`, bookID, authorID); err == nil {
		t.Fatal("expected relationship metadata to be required")
	}
	if _, err := db.Exec(`INSERT INTO book_authors (book_id, author_id, created_at, updated_at) VALUES (?, ?, ?, ?)`, bookID, authorID, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO book_authors (book_id, author_id, created_at, updated_at) VALUES (?, ?, ?, ?)`, bookID, authorID, now, now); err == nil {
		t.Fatal("expected relationship foreign-key pair to be unique")
	}
}

func openConnection(t *testing.T, ctx context.Context, db *sql.DB) *sql.Conn {
	t.Helper()

	conn, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	return conn
}

func assertForeignKeysEnabled(t *testing.T, ctx context.Context, conn *sql.Conn) {
	t.Helper()

	var enabled int
	if err := conn.QueryRowContext(ctx, "PRAGMA foreign_keys").Scan(&enabled); err != nil {
		t.Fatal(err)
	}
	if enabled != 1 {
		t.Fatalf("expected foreign keys enabled, got %d", enabled)
	}
}
