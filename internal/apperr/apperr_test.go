package apperr_test

import (
	"errors"
	"fmt"
	"testing"

	"bakku.dev/bookist/internal/apperr"
)

var errSentinel = apperr.Invalid("invalid_thing", "thing is invalid", "thing")

func TestAsFindsWrappedError(t *testing.T) {
	wrapped := fmt.Errorf("while saving: %w", errSentinel)

	got, ok := apperr.As(wrapped)
	if !ok {
		t.Fatal("expected to find an apperr.Error")
	}
	if got.Kind != apperr.KindInvalid || got.Code != "invalid_thing" || got.Message != "thing is invalid" || got.Field != "thing" {
		t.Fatalf("unexpected error: %#v", got)
	}
}

func TestAsIgnoresOtherErrors(t *testing.T) {
	if _, ok := apperr.As(errors.New("boom")); ok {
		t.Fatal("expected no apperr.Error")
	}
}

func TestSentinelKeepsErrorsIsIdentity(t *testing.T) {
	wrapped := fmt.Errorf("while saving: %w", errSentinel)
	if !errors.Is(wrapped, errSentinel) {
		t.Fatal("expected errors.Is to match the sentinel")
	}
	if errors.Is(wrapped, apperr.Invalid("invalid_thing", "thing is invalid", "thing")) {
		t.Fatal("expected a different value with the same code not to match")
	}
	if errSentinel.Error() != "thing is invalid" {
		t.Fatalf("unexpected message %q", errSentinel.Error())
	}
}
