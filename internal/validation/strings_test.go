package validation_test

import (
	"testing"

	"bakku.dev/bookist/internal/validation"
)

func TestTrimToNil(t *testing.T) {
	for _, test := range []struct {
		name  string
		value *string
		want  *string
	}{
		{name: "nil", value: nil, want: nil},
		{name: "empty", value: new(""), want: nil},
		{name: "whitespace only", value: new(" \t\n"), want: nil},
		{name: "trimmed", value: new("  Dune  "), want: new("Dune")},
		{name: "unchanged", value: new("Dune"), want: new("Dune")},
	} {
		t.Run(test.name, func(t *testing.T) {
			got := validation.TrimToNil(test.value)
			if (got == nil) != (test.want == nil) || (got != nil && *got != *test.want) {
				t.Fatalf("expected %v, got %v", test.want, got)
			}
		})
	}
}
