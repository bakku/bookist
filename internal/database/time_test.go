package database_test

import (
	"regexp"
	"testing"
	"time"

	"bakku.dev/bookist/internal/database"
)

// ── Format ────────────────────────────────────────────────────────────────────

func TestFormatTimeIsFixedWidthMillisecondsInUTC(t *testing.T) {
	zone := time.FixedZone("CEST", 2*60*60)

	tests := []struct {
		name string
		in   time.Time
		want string
	}{
		{name: "whole second keeps zero fraction", in: time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC), want: "2026-01-02T03:04:05.000Z"},
		{name: "trailing zeros kept", in: time.Date(2026, 1, 2, 3, 4, 5, 120_000_000, time.UTC), want: "2026-01-02T03:04:05.120Z"},
		{name: "sub-millisecond truncated", in: time.Date(2026, 1, 2, 3, 4, 5, 999_999_999, time.UTC), want: "2026-01-02T03:04:05.999Z"},
		{name: "converted to UTC", in: time.Date(2026, 1, 2, 5, 4, 5, 7_000_000, zone), want: "2026-01-02T03:04:05.007Z"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := database.FormatTime(test.in); got != test.want {
				t.Fatalf("expected %q, got %q", test.want, got)
			}
		})
	}
}

func TestFormatTimeSortsChronologicallyAsText(t *testing.T) {
	earlier := database.FormatTime(time.Date(2026, 1, 2, 3, 4, 5, 90_000_000, time.UTC))
	later := database.FormatTime(time.Date(2026, 1, 2, 3, 4, 5, 100_000_000, time.UTC))

	if earlier >= later {
		t.Fatalf("expected %q to sort before %q", earlier, later)
	}
}

func TestNowUsesStoredLayout(t *testing.T) {
	pattern := regexp.MustCompile(`^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}:\d{2}\.\d{3}Z$`)

	if got := database.Now(); !pattern.MatchString(got) {
		t.Fatalf("expected fixed-width UTC timestamp, got %q", got)
	}
}

// ── Parse ─────────────────────────────────────────────────────────────────────

func TestParseTimeReadsStoredAndSecondPrecisionTimestamps(t *testing.T) {
	want := time.Date(2026, 1, 2, 3, 4, 5, 120_000_000, time.UTC)

	got, err := database.ParseTime(database.FormatTime(want))
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(want) {
		t.Fatalf("expected %s, got %s", want, got)
	}

	got, err = database.ParseTime("2026-01-02T03:04:05Z")
	if err != nil {
		t.Fatal(err)
	}
	if !got.Equal(want.Truncate(time.Second)) {
		t.Fatalf("expected %s, got %s", want.Truncate(time.Second), got)
	}
}

func TestTimeScannerParsesTextAndRejectsInvalidValues(t *testing.T) {
	var got time.Time
	scanner := database.Time(&got)

	if err := scanner.Scan("2026-01-02T03:04:05.120Z"); err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, 1, 2, 3, 4, 5, 120_000_000, time.UTC); !got.Equal(want) {
		t.Fatalf("expected %s, got %s", want, got)
	}

	if err := scanner.Scan([]byte("2026-01-02T03:04:06.000Z")); err != nil {
		t.Fatal(err)
	}
	if want := time.Date(2026, 1, 2, 3, 4, 6, 0, time.UTC); !got.Equal(want) {
		t.Fatalf("expected %s, got %s", want, got)
	}

	if err := scanner.Scan("yesterday"); err == nil {
		t.Fatal("expected error for an unparseable timestamp")
	}
	if err := scanner.Scan(nil); err == nil {
		t.Fatal("expected error for a NULL timestamp")
	}
}
