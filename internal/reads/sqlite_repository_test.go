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
	finishedAt := "2026-01-03"
	rating := 4.5
	notes := "Excellent"

	created, err := repository.Create(context.Background(), bookID, reads.CreateReadRequest{
		StartedAt: &startedAt, FinishedAt: &finishedAt, Rating: &rating, Notes: &notes,
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
	var gotStartedAt, gotFinishedAt, gotNotes sql.NullString
	var gotRating sql.NullFloat64
	var createdAt, updatedAt string
	err = db.QueryRow(`
		SELECT book_id, started_at, finished_at, rating, notes, created_at, updated_at
		FROM reads WHERE id = ?
	`, created.ID).Scan(&gotBookID, &gotStartedAt, &gotFinishedAt, &gotRating, &gotNotes, &createdAt, &updatedAt)
	if err != nil {
		t.Fatal(err)
	}
	if gotBookID != bookID || gotStartedAt.String != startedAt || gotFinishedAt.String != finishedAt || gotRating.Float64 != rating || gotNotes.String != notes {
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
	testsupport.InsertReadRow(t, db, testsupport.ReadRow{
		ID: "old", BookID: bookID, StartedAt: new("2025-01-01"), FinishedAt: new("2025-01-02"),
		CreatedAt: "2026-01-01T00:00:00Z",
	})
	testsupport.InsertReadRow(t, db, testsupport.ReadRow{
		ID: "new", BookID: bookID, StartedAt: new("2026-01-01"), FinishedAt: new("2026-01-02"),
		CreatedAt: "2026-01-02T00:00:00Z",
	})
	testsupport.InsertReadRow(t, db, testsupport.ReadRow{
		ID: "other", BookID: otherBookID, StartedAt: new("2027-01-01"), FinishedAt: new("2027-01-02"),
		CreatedAt: "2027-01-02T00:00:00Z",
	})

	listed, err := reads.NewSQLiteRepository(db).ListByBookID(context.Background(), bookID)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 2 || listed[0].ID != "new" || listed[1].ID != "old" {
		t.Fatalf("expected newest reads first, got %#v", listed)
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
		name       string
		startedAt  string
		finishedAt string
		rating     float64
	}{
		{name: "rating range", startedAt: "2026-01-01", finishedAt: "2026-01-02", rating: 5.5},
		{name: "rating increment", startedAt: "2026-01-01", finishedAt: "2026-01-02", rating: 4.2},
		{name: "date order", startedAt: "2026-01-02", finishedAt: "2026-01-01", rating: 4.5},
	} {
		t.Run(test.name, func(t *testing.T) {
			_, err := db.Exec(`
				INSERT INTO reads (id, book_id, started_at, finished_at, rating, created_at, updated_at)
				VALUES (?, ?, ?, ?, ?, ?, ?)
			`, uuid.NewString(), bookID, test.startedAt, test.finishedAt, test.rating, now, now)
			if err == nil {
				t.Fatal("expected database constraint error")
			}
		})
	}
}
