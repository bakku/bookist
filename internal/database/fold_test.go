package database_test

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"bakku.dev/bookist/internal/database"
)

// ── Fold ──────────────────────────────────────────────────────────────────────

func TestFold(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  string
	}{
		{name: "ascii", input: "Dune", want: "dune"},
		{name: "empty", input: "", want: ""},
		{name: "non-ASCII case", input: "ÜBER Menschen", want: "uber menschen"},
		{name: "accents", input: "Café Crème", want: "cafe creme"},
		{name: "decomposed accents", input: "Café", want: "cafe"},
		{name: "sharp s", input: "Straße", want: "strasse"},
		{name: "greek final sigma", input: "ΟΔΟΣ", want: "οδοσ"},
		{name: "wildcards untouched", input: "100%_", want: "100%_"},
		{name: "unaccented letters kept", input: "Łódź Øl", want: "łodz øl"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := database.Fold(test.input); got != test.want {
				t.Fatalf("expected %q, got %q", test.want, got)
			}
		})
	}
}

func TestFoldIsIdempotent(t *testing.T) {
	for _, input := range []string{"Über", "Straße", "ΟΔΟΣ", "Café", "plain"} {
		once := database.Fold(input)
		if twice := database.Fold(once); twice != once {
			t.Fatalf("expected Fold(%q) to be stable, got %q then %q", input, once, twice)
		}
	}
}

// ── bookist_fold ──────────────────────────────────────────────────────────────

func TestOpenRegistersFoldFunctionOnEveryConnection(t *testing.T) {
	ctx := context.Background()

	db, err := database.Open(ctx, filepath.Join(t.TempDir(), "bookist.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	first, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = first.Close() })
	second, err := db.Conn(ctx)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = second.Close() })

	for _, conn := range []*sql.Conn{first, second} {
		var folded string
		if err := conn.QueryRowContext(ctx, `SELECT bookist_fold('ÜBER Straße')`).Scan(&folded); err != nil {
			t.Fatal(err)
		}
		if folded != "uber strasse" {
			t.Fatalf("expected %q, got %q", "uber strasse", folded)
		}
	}
}

func TestFoldFunctionReturnsNullForNull(t *testing.T) {
	ctx := context.Background()

	db, err := database.Open(ctx, filepath.Join(t.TempDir(), "bookist.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	var folded sql.NullString
	if err := db.QueryRowContext(ctx, `SELECT bookist_fold(NULL)`).Scan(&folded); err != nil {
		t.Fatal(err)
	}
	if folded.Valid {
		t.Fatalf("expected NULL, got %q", folded.String)
	}
}
