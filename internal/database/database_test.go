package database_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"bakku.dev/bookist/internal/database"
	"bakku.dev/bookist/internal/testsupport"
	"github.com/google/uuid"
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
					notNull bool
					primary bool
				}{notNull: notNull == 1, primary: primary == 1}
			}
			if !columns["id"].primary {
				t.Fatal("expected id primary key")
			}
			if !columns["created_at"].notNull || !columns["updated_at"].notNull {
				t.Fatal("expected non-null created_at and updated_at")
			}
		})
	}
}

func TestInitialSchemaRejectsNonUUIDIDs(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	if _, err := db.Exec(`INSERT INTO authors (id, name, created_at, updated_at) VALUES ('not-a-uuid', 'Jane Austen', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`); err == nil {
		t.Fatal("expected non-UUID ID to violate a constraint")
	}
}

func TestInitialSchemaEnforcesNaturalKeysAndBookValidation(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	now := "2026-01-02T03:04:05Z"

	if _, err := db.Exec(`INSERT INTO authors (id, name, created_at, updated_at) VALUES (?, 'Jane Austen', ?, ?)`, uuid.NewString(), now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO authors (id, name, created_at, updated_at) VALUES (?, ?, ?, ?)`, uuid.NewString(), "jane AUSTEN", now, now); err == nil {
		t.Fatal("expected case-insensitive duplicate author name to violate a constraint")
	}

	for _, test := range []struct {
		title string
		cols  string
		args  []any
	}{
		{title: "invalid pages", cols: "pages", args: []any{0}},
		{title: "month without year", cols: "published_month", args: []any{1}},
		{title: "invalid calendar day", cols: "published_year, published_month, published_day", args: []any{2023, 2, 29}},
		{title: "invalid purchased date", cols: "purchased_at", args: []any{"2025-02-29"}},
	} {
		placeholders := "?"
		for range test.args[1:] {
			placeholders += ", ?"
		}
		args := []any{uuid.NewString(), test.title}
		args = append(args, test.args...)
		args = append(args, now, now)
		query := `INSERT INTO books (id, title, ` + test.cols + `, created_at, updated_at) VALUES (?, ?, ` + placeholders + `, ?, ?)`
		if _, err := db.Exec(query, args...); err == nil {
			t.Fatalf("expected %s to violate a constraint", test.title)
		}
	}
	if _, err := db.Exec(`INSERT INTO books (id, title, created_at, updated_at) VALUES (?, 'Dune', ?, ?)`, uuid.NewString(), now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO books (id, title, created_at, updated_at) VALUES (?, 'dUnE', ?, ?)`, uuid.NewString(), now, now); err == nil {
		t.Fatal("expected case-insensitive duplicate title to violate a constraint")
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
	if _, err := db.Exec(`INSERT INTO book_authors (id, book_id, author_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`, uuid.NewString(), bookID, authorID, now, now); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`INSERT INTO book_authors (id, book_id, author_id, created_at, updated_at) VALUES (?, ?, ?, ?, ?)`, uuid.NewString(), bookID, authorID, now, now); err == nil {
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
