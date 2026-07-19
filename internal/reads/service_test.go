package reads_test

import (
	"context"
	"errors"
	"testing"

	"bakku.dev/bookist/internal/reads"
	"bakku.dev/bookist/internal/testsupport"
)

// ── Create ────────────────────────────────────────────────────────────────────

func TestServiceCreateNormalizesInput(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)
	service := reads.NewService(reads.NewSQLiteRepository(db))
	startedAt := " 2026-01-02 "
	abandonedAt := " 2026-01-03 "
	rating := 4.5
	notes := "  Excellent  "

	created, err := service.Create(context.Background(), bookID, reads.CreateReadRequest{
		StartedAt: &startedAt, AbandonedAt: &abandonedAt, Rating: &rating, Notes: &notes,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.StartedAt == nil || *created.StartedAt != "2026-01-02" {
		t.Fatalf("expected normalized started_at, got %#v", created.StartedAt)
	}
	if created.AbandonedAt == nil || *created.AbandonedAt != "2026-01-03" {
		t.Fatalf("expected normalized abandoned_at, got %#v", created.AbandonedAt)
	}
	if created.Notes == nil || *created.Notes != "Excellent" {
		t.Fatalf("expected normalized notes, got %#v", created.Notes)
	}
}

func TestServiceCreateNormalizesEmptyOptionalStringsToNull(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)
	service := reads.NewService(reads.NewSQLiteRepository(db))
	empty := "  "

	created, err := service.Create(context.Background(), bookID, reads.CreateReadRequest{
		StartedAt: &empty, FinishedAt: &empty, AbandonedAt: &empty, Notes: &empty,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.StartedAt != nil || created.FinishedAt != nil || created.AbandonedAt != nil || created.Notes != nil {
		t.Fatalf("expected empty optional values to become nil, got %#v", created)
	}
}

func TestServiceCreateRejectsInvalidInput(t *testing.T) {
	tests := []struct {
		name  string
		input reads.CreateReadRequest
		want  error
	}{
		{name: "invalid started date", input: reads.CreateReadRequest{StartedAt: new("01/02/2026")}, want: reads.ErrInvalidStartedAt},
		{name: "invalid finished date", input: reads.CreateReadRequest{FinishedAt: new("2026-02-30")}, want: reads.ErrInvalidFinishedAt},
		{name: "invalid abandoned date", input: reads.CreateReadRequest{AbandonedAt: new("2026-02-30")}, want: reads.ErrInvalidAbandonedAt},
		{name: "finished before started", input: reads.CreateReadRequest{StartedAt: new("2026-02-02"), FinishedAt: new("2026-02-01")}, want: reads.ErrFinishedBeforeStarted},
		{name: "abandoned before started", input: reads.CreateReadRequest{StartedAt: new("2026-02-02"), AbandonedAt: new("2026-02-01")}, want: reads.ErrAbandonedBeforeStarted},
		{name: "conflicting terminal dates", input: reads.CreateReadRequest{FinishedAt: new("2026-02-02"), AbandonedAt: new("2026-02-03")}, want: reads.ErrConflictingTerminalDates},
		{name: "rating below range", input: reads.CreateReadRequest{Rating: new(0.5)}, want: reads.ErrInvalidRating},
		{name: "rating above range", input: reads.CreateReadRequest{Rating: new(5.5)}, want: reads.ErrInvalidRating},
		{name: "rating wrong increment", input: reads.CreateReadRequest{Rating: new(4.2)}, want: reads.ErrInvalidRating},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			db := testsupport.OpenMigratedDB(t)
			bookID := testsupport.InsertBookRow(t, db, "Dune", nil)
			service := reads.NewService(reads.NewSQLiteRepository(db))

			_, err := service.Create(context.Background(), bookID, test.input)
			if !errors.Is(err, test.want) {
				t.Fatalf("expected %v, got %v", test.want, err)
			}

			var count int
			if err := db.QueryRow(`SELECT COUNT(*) FROM reads`).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Fatalf("expected no persisted reads, got %d", count)
			}
		})
	}
}

func TestServiceCreatePermitsAbandonedReadWithoutStart(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)
	abandonedAt := "2026-01-03"
	rating := 4.5
	notes := "Unfinished, but insightful"

	created, err := reads.NewService(reads.NewSQLiteRepository(db)).Create(context.Background(), bookID, reads.CreateReadRequest{
		AbandonedAt: &abandonedAt, Rating: &rating, Notes: &notes,
	})
	if err != nil {
		t.Fatal(err)
	}
	if created.StartedAt != nil || created.AbandonedAt == nil || *created.AbandonedAt != abandonedAt ||
		created.Rating == nil || *created.Rating != rating || created.Notes == nil || *created.Notes != notes {
		t.Fatalf("unexpected abandoned read: %#v", created)
	}
}

// ── ListByBookID ──────────────────────────────────────────────────────────────

func TestServiceListByBookIDReturnsReads(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)
	readID := int64(100)
	testsupport.InsertReadRow(t, db, testsupport.ReadRow{
		ID: readID, BookID: bookID, StartedAt: new("2026-01-01"),
		AbandonedAt: new("2026-01-03"), Rating: new(4.5), Notes: new("Excellent"),
		CreatedAt: "2026-01-04T00:00:00Z",
	})
	service := reads.NewService(reads.NewSQLiteRepository(db))

	listed, err := service.ListByBookID(context.Background(), bookID)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected 1 read, got %d", len(listed))
	}
	if listed[0].ID != readID || listed[0].AbandonedAt == nil || *listed[0].AbandonedAt != "2026-01-03" || listed[0].Rating == nil || *listed[0].Rating != 4.5 {
		t.Fatalf("unexpected read: %#v", listed[0])
	}
}
