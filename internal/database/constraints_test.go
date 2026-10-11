package database_test

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"
	"testing"

	"bakku.dev/bookist/internal/database"
)

// ── Constraint Errors ─────────────────────────────────────────────────────────

func TestConstraintHelpersClassifySQLiteErrors(t *testing.T) {
	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "constraints.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`
		CREATE TABLE parents (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL UNIQUE
		);
		CREATE TABLE children (
			id INTEGER PRIMARY KEY,
			parent_id INTEGER NULL REFERENCES parents(id),
			label TEXT NULL CHECK (label IS NULL OR label <> ''),
			rating INTEGER NULL CONSTRAINT children_rating_valid CHECK (rating IS NULL OR rating > 0)
		);
		INSERT INTO parents (id, name) VALUES (1, 'taken');
	`); err != nil {
		t.Fatal(err)
	}

	exec := func(query string) error {
		_, err := db.Exec(query)
		if err == nil {
			t.Fatalf("expected %q to fail", query)
		}
		return err
	}
	uniqueErr := exec(`INSERT INTO parents (name) VALUES ('taken')`)

	tests := []struct {
		name       string
		err        error
		unique     bool
		foreignKey bool
		check      string
	}{
		{name: "unique", err: uniqueErr, unique: true},
		{name: "wrapped unique", err: fmt.Errorf("insert parent: %w", uniqueErr), unique: true},
		{name: "foreign key", err: exec(`INSERT INTO children (parent_id) VALUES (999)`), foreignKey: true},
		{name: "named check", err: exec(`INSERT INTO children (rating) VALUES (0)`), check: "children_rating_valid"},
		{name: "unnamed check", err: exec(`INSERT INTO children (label) VALUES ('')`)},
		{name: "not null", err: exec(`INSERT INTO parents (name) VALUES (NULL)`)},
		{name: "non-sqlite error with constraint text", err: errors.New("UNIQUE constraint failed: parents.name")},
		{name: "nil", err: nil},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := database.IsUniqueViolation(tt.err); got != tt.unique {
				t.Fatalf("IsUniqueViolation(%v) = %t, want %t", tt.err, got, tt.unique)
			}
			if got := database.IsForeignKeyViolation(tt.err); got != tt.foreignKey {
				t.Fatalf("IsForeignKeyViolation(%v) = %t, want %t", tt.err, got, tt.foreignKey)
			}
			for _, constraintName := range []string{"children_rating_valid", "children_rating", "children_rating_valid_strict"} {
				want := constraintName == tt.check
				if got := database.IsCheckViolation(tt.err, constraintName); got != want {
					t.Fatalf("IsCheckViolation(%v, %q) = %t, want %t", tt.err, constraintName, got, want)
				}
			}
		})
	}
}
