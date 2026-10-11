package database_test

import (
	"errors"
	"fmt"
	"strings"
	"testing"

	"bakku.dev/bookist/internal/database"
	"bakku.dev/bookist/internal/testsupport"

	sqlite3 "modernc.org/sqlite/lib"
)

const ratingConstraint = "reads_rating_valid"

// ── Constraint Classification ──────────────────────────────────────────────────

func TestConstraintCodes(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)
	testsupport.InsertListRow(t, db, "Favorites")

	for _, test := range []struct {
		name       string
		query      string
		args       []any
		unique     bool
		foreignKey bool
		check      bool
	}{
		{
			name: "unique", unique: true,
			query: `INSERT INTO lists (name, created_at, updated_at) VALUES ('favorites', '', '')`,
		},
		{
			name: "foreign key", foreignKey: true,
			query: `INSERT INTO reads (book_id, created_at, updated_at) VALUES (?, '', '')`,
			args:  []any{bookID + 1},
		},
		{
			name: "named check", check: true,
			query: `INSERT INTO reads (book_id, rating, created_at, updated_at) VALUES (?, 4.2, '', '')`,
			args:  []any{bookID},
		},
		{
			name:  "not null",
			query: `INSERT INTO books (title, created_at, updated_at) VALUES (NULL, '', '')`,
		},
		{
			name:  "primary key",
			query: `INSERT INTO books (id, title, created_at, updated_at) VALUES (?, 'Duplicate', '', '')`,
			args:  []any{bookID},
		},
		{
			name:  "unrelated database error",
			query: `INSERT INTO missing_table DEFAULT VALUES`,
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := db.Exec(test.query, test.args...)
			if err == nil {
				t.Fatal("expected SQLite error")
			}

			// Wrapping must preserve classification.
			for _, candidate := range []error{err, fmt.Errorf("write: %w", err)} {
				if got := database.IsUniqueViolation(candidate); got != test.unique {
					t.Fatalf("unique: got %t, want %t (%v)", got, test.unique, candidate)
				}
				if got := database.IsForeignKeyViolation(candidate); got != test.foreignKey {
					t.Fatalf("foreign key: got %t, want %t (%v)", got, test.foreignKey, candidate)
				}
				if got := database.IsCheckViolation(candidate, ratingConstraint); got != test.check {
					t.Fatalf("check: got %t, want %t (%v)", got, test.check, candidate)
				}
				for _, name := range []string{"", strings.TrimSuffix(ratingConstraint, "_valid"), ratingConstraint + "_other"} {
					if database.IsCheckViolation(candidate, name) {
						t.Fatalf("unexpected check match for %q: %v", name, candidate)
					}
				}
			}
		})
	}
}

func TestConstraintTextIgnored(t *testing.T) {
	for _, err := range []error{
		nil,
		errors.New("UNIQUE constraint failed"),
		errors.New("FOREIGN KEY constraint failed"),
		fmt.Errorf("CHECK constraint failed: %s (%d)", ratingConstraint, sqlite3.SQLITE_CONSTRAINT_CHECK),
	} {
		if database.IsUniqueViolation(err) || database.IsForeignKeyViolation(err) || database.IsCheckViolation(err, ratingConstraint) {
			t.Fatalf("unexpected constraint match: %v", err)
		}
	}
}
