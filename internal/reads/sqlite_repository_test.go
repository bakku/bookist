package reads_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"bakku.dev/bookist/internal/reads"
	"bakku.dev/bookist/internal/testsupport"
	"github.com/google/uuid"
)

// ── Create ────────────────────────────────────────────────────────────────────

func TestSQLiteRepositoryCreatePersistsRead(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)
	repository := reads.NewSQLiteRepository(db)
	startedAt := "2026-01-01"
	abandonedAt := "2026-01-03"
	rating := 4.5
	notes := "Excellent"

	created, err := repository.Create(context.Background(), bookID, reads.CreateReadRequest{
		StartedAt: &startedAt, AbandonedAt: &abandonedAt, Rating: &rating, Notes: &notes,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := uuid.Parse(created.ID); err != nil {
		t.Fatalf("expected UUID read ID, got %q", created.ID)
	}
	if created.CreatedAt.IsZero() || created.UpdatedAt.IsZero() {
		t.Fatal("expected backend timestamps")
	}

	var gotBookID string
	var gotStartedAt, gotFinishedAt, gotAbandonedAt, gotNotes sql.NullString
	var gotRating sql.NullFloat64
	var createdAt, updatedAt string
	err = db.QueryRow(`
		SELECT book_id, started_at, finished_at, abandoned_at, rating, notes, created_at, updated_at
		FROM reads WHERE id = ?
	`, created.ID).Scan(&gotBookID, &gotStartedAt, &gotFinishedAt, &gotAbandonedAt, &gotRating, &gotNotes, &createdAt, &updatedAt)
	if err != nil {
		t.Fatal(err)
	}
	if gotBookID != bookID || gotStartedAt.String != startedAt || gotFinishedAt.Valid || gotAbandonedAt.String != abandonedAt || gotRating.Float64 != rating || gotNotes.String != notes {
		t.Fatalf("unexpected persisted read values")
	}
	if _, err := time.Parse(time.RFC3339, createdAt); err != nil {
		t.Fatalf("invalid created_at: %v", err)
	}
	if _, err := time.Parse(time.RFC3339, updatedAt); err != nil {
		t.Fatalf("invalid updated_at: %v", err)
	}
}

func TestSQLiteRepositoryCreateReturnsBookNotFound(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	repository := reads.NewSQLiteRepository(db)

	_, err := repository.Create(context.Background(), uuid.NewString(), reads.CreateReadRequest{})
	if !errors.Is(err, reads.ErrBookNotFound) {
		t.Fatalf("expected ErrBookNotFound, got %v", err)
	}
}

// ── ListByBookID ──────────────────────────────────────────────────────────────

func TestSQLiteRepositoryListByBookIDOrdersNewestFirst(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)
	otherBookID := testsupport.InsertBookRow(t, db, "Foundation", nil)
	oldID := uuid.NewString()
	newID := uuid.NewString()
	testsupport.InsertReadRow(t, db, testsupport.ReadRow{
		ID: oldID, BookID: bookID, StartedAt: new("2025-01-01"), FinishedAt: new("2025-01-02"),
		CreatedAt: "2026-01-01T00:00:00Z",
	})
	testsupport.InsertReadRow(t, db, testsupport.ReadRow{
		ID: newID, BookID: bookID, StartedAt: new("2026-01-01"), AbandonedAt: new("2026-01-02"),
		CreatedAt: "2026-01-02T00:00:00Z",
	})
	testsupport.InsertReadRow(t, db, testsupport.ReadRow{
		ID: uuid.NewString(), BookID: otherBookID, StartedAt: new("2027-01-01"), FinishedAt: new("2027-01-02"),
		CreatedAt: "2027-01-02T00:00:00Z",
	})

	listed, err := reads.NewSQLiteRepository(db).ListByBookID(context.Background(), bookID)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 2 || listed[0].ID != newID || listed[1].ID != oldID {
		t.Fatalf("expected newest reads first, got %#v", listed)
	}
	if listed[0].AbandonedAt == nil || *listed[0].AbandonedAt != "2026-01-02" || listed[0].FinishedAt != nil {
		t.Fatalf("expected abandoned read, got %#v", listed[0])
	}
}

func TestSQLiteRepositoryListByBookIDReturnsEmptySlice(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)

	listed, err := reads.NewSQLiteRepository(db).ListByBookID(context.Background(), bookID)
	if err != nil {
		t.Fatal(err)
	}
	if listed == nil || len(listed) != 0 {
		t.Fatalf("expected empty non-nil slice, got %#v", listed)
	}
}

func TestSQLiteRepositoryListByBookIDReturnsBookNotFound(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)

	_, err := reads.NewSQLiteRepository(db).ListByBookID(context.Background(), uuid.NewString())
	if !errors.Is(err, reads.ErrBookNotFound) {
		t.Fatalf("expected ErrBookNotFound, got %v", err)
	}
}

// ── Constraints ───────────────────────────────────────────────────────────────

func TestReadsTableEnforcesRatingAndDateConstraints(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)
	now := "2026-01-01T00:00:00Z"

	for _, test := range []struct {
		name        string
		startedAt   *string
		finishedAt  *string
		abandonedAt *string
		rating      *float64
	}{
		{name: "rating range", startedAt: new("2026-01-01"), finishedAt: new("2026-01-02"), rating: new(5.5)},
		{name: "rating increment", startedAt: new("2026-01-01"), finishedAt: new("2026-01-02"), rating: new(4.2)},
		{name: "finished date order", startedAt: new("2026-01-02"), finishedAt: new("2026-01-01"), rating: new(4.5)},
		{name: "abandoned date order", startedAt: new("2026-01-02"), abandonedAt: new("2026-01-01")},
		{name: "conflicting terminal dates", finishedAt: new("2026-01-02"), abandonedAt: new("2026-01-03")},
		{name: "invalid calendar date", startedAt: new("2026-02-30"), finishedAt: new("2026-03-01"), rating: new(4.5)},
		{name: "invalid abandoned calendar date", abandonedAt: new("2026-02-30")},
		{name: "abandoned year zero", abandonedAt: new("0000-01-01")},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := db.Exec(`
				INSERT INTO reads (id, book_id, started_at, finished_at, abandoned_at, rating, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?, ?)
			`, uuid.NewString(), bookID, test.startedAt, test.finishedAt, test.abandonedAt, test.rating, now, now)
			if err == nil {
				t.Fatal("expected database constraint error")
			}
		})
	}
}

func TestReadsTablePermitsPartialAndSameDayTerminalDates(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)
	now := "2026-01-01T00:00:00Z"

	for _, test := range []struct {
		name        string
		startedAt   *string
		finishedAt  *string
		abandonedAt *string
	}{
		{name: "finished without start", finishedAt: new("2026-01-02")},
		{name: "abandoned without start", abandonedAt: new("2026-01-02")},
		{name: "abandoned on start date", startedAt: new("2026-01-02"), abandonedAt: new("2026-01-02")},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := db.Exec(`
				INSERT INTO reads (id, book_id, started_at, finished_at, abandoned_at, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`, uuid.NewString(), bookID, test.startedAt, test.finishedAt, test.abandonedAt, now, now)
			if err != nil {
				t.Fatalf("expected valid read: %v", err)
			}
		})
	}
}
