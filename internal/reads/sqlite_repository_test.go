package reads_test

import (
	"context"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"bakku.dev/bookist/internal/database"
	"bakku.dev/bookist/internal/optional"
	"bakku.dev/bookist/internal/reads"
	"bakku.dev/bookist/internal/testsupport"
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
	if created.ID <= 0 {
		t.Fatalf("expected positive read ID, got %d", created.ID)
	}
	if created.CreatedAt.IsZero() || created.UpdatedAt.IsZero() {
		t.Fatal("expected backend timestamps")
	}

	testsupport.AssertReadRow(t, db, created.ID, testsupport.ReadRowAssertion{
		BookID:      bookID,
		StartedAt:   &startedAt,
		AbandonedAt: &abandonedAt,
		Rating:      &rating,
		Notes:       &notes,
	})
}

func TestSQLiteRepositoryCreateReturnsBookNotFound(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	repository := reads.NewSQLiteRepository(db)

	_, err := repository.Create(context.Background(), 999999, reads.CreateReadRequest{})
	if !errors.Is(err, reads.ErrBookNotFound) {
		t.Fatalf("expected ErrBookNotFound, got %v", err)
	}
}

// ── GetByID ───────────────────────────────────────────────────────────────────

func TestSQLiteRepositoryGetByIDReturnsPersistedRead(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)
	testsupport.InsertReadRow(t, db, testsupport.ReadRow{
		ID: 100, BookID: bookID, FinishedAt: new("2026-01-03"), Rating: new(5.0),
		CreatedAt: "2026-01-04T00:00:00Z",
	})

	got, err := reads.NewSQLiteRepository(db).GetByID(context.Background(), 100)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != 100 || got.BookID != bookID || got.FinishedAt == nil || *got.FinishedAt != "2026-01-03" {
		t.Fatalf("unexpected read: %#v", got)
	}
}

func TestSQLiteRepositoryGetByIDReturnsReadNotFound(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)

	_, err := reads.NewSQLiteRepository(db).GetByID(context.Background(), 999999)
	if !errors.Is(err, reads.ErrReadNotFound) {
		t.Fatalf("expected ErrReadNotFound, got %v", err)
	}
}

// ── ListByBookID ──────────────────────────────────────────────────────────────

func TestSQLiteRepositoryListByBookIDOrdersNewestFirst(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)
	otherBookID := testsupport.InsertBookRow(t, db, "Foundation", nil)
	oldID := int64(100)
	newID := int64(101)
	testsupport.InsertReadRow(t, db, testsupport.ReadRow{
		ID: oldID, BookID: bookID, StartedAt: new("2025-01-01"), FinishedAt: new("2025-01-02"),
		CreatedAt: "2026-01-01T00:00:00Z",
	})
	testsupport.InsertReadRow(t, db, testsupport.ReadRow{
		ID: newID, BookID: bookID, StartedAt: new("2026-01-01"), AbandonedAt: new("2026-01-02"),
		CreatedAt: "2026-01-02T00:00:00Z",
	})
	testsupport.InsertReadRow(t, db, testsupport.ReadRow{
		ID: 102, BookID: otherBookID, StartedAt: new("2027-01-01"), FinishedAt: new("2027-01-02"),
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

	_, err := reads.NewSQLiteRepository(db).ListByBookID(context.Background(), 999999)
	if !errors.Is(err, reads.ErrBookNotFound) {
		t.Fatalf("expected ErrBookNotFound, got %v", err)
	}
}

// ── Update ────────────────────────────────────────────────────────────────────

func TestSQLiteRepositoryUpdateOnlyChangesPresentFields(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)
	readID := int64(100)
	createdAt := "2026-01-01T00:00:00Z"
	testsupport.InsertReadRow(t, db, testsupport.ReadRow{
		ID: readID, BookID: bookID, StartedAt: new("2026-01-01"), FinishedAt: new("2026-01-02"),
		Rating: new(4.5), Notes: new("Original"), CreatedAt: createdAt,
	})

	updated, err := reads.NewSQLiteRepository(db).Update(context.Background(), readID, reads.UpdateReadRequest{
		FinishedAt: optional.Value[string]{Present: true},
		Notes:      optional.Value[string]{Present: true, Value: new("Revised")},
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.BookID != bookID || updated.CreatedAt.Format(time.RFC3339) != createdAt {
		t.Fatalf("book_id or created_at changed: %#v", updated)
	}
	if updated.FinishedAt != nil || updated.Notes == nil || *updated.Notes != "Revised" {
		t.Fatalf("explicit fields not updated: %#v", updated)
	}
	if updated.StartedAt == nil || *updated.StartedAt != "2026-01-01" || updated.Rating == nil || *updated.Rating != 4.5 {
		t.Fatalf("omitted fields changed: %#v", updated)
	}
	if !updated.UpdatedAt.After(updated.CreatedAt) {
		t.Fatalf("expected updated_at to advance, got %s", updated.UpdatedAt)
	}
}

func TestSQLiteRepositoryUpdateReturnsReadNotFound(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)

	_, err := reads.NewSQLiteRepository(db).Update(context.Background(), 999999, reads.UpdateReadRequest{
		Notes: optional.Value[string]{Present: true, Value: new("Missing")},
	})
	if !errors.Is(err, reads.ErrReadNotFound) {
		t.Fatalf("expected ErrReadNotFound, got %v", err)
	}
}

func TestSQLiteRepositoryUpdateTranslatesNamedCheckConstraints(t *testing.T) {
	tests := []struct {
		name  string
		input reads.UpdateReadRequest
		want  error
	}{
		{
			name:  "started_at format",
			input: reads.UpdateReadRequest{StartedAt: optional.Value[string]{Present: true, Value: new("not-a-date")}},
			want:  reads.ErrInvalidStartedAt,
		},
		{
			name:  "finished_at format",
			input: reads.UpdateReadRequest{FinishedAt: optional.Value[string]{Present: true, Value: new("not-a-date")}},
			want:  reads.ErrInvalidFinishedAt,
		},
		{
			name:  "abandoned_at format",
			input: reads.UpdateReadRequest{AbandonedAt: optional.Value[string]{Present: true, Value: new("not-a-date")}},
			want:  reads.ErrInvalidAbandonedAt,
		},
		{
			name:  "rating range and increment",
			input: reads.UpdateReadRequest{Rating: optional.Value[float64]{Present: true, Value: new(4.2)}},
			want:  reads.ErrInvalidRating,
		},
		{
			name: "terminal dates are mutually exclusive",
			input: reads.UpdateReadRequest{
				FinishedAt:  optional.Value[string]{Present: true, Value: new("2026-02-02")},
				AbandonedAt: optional.Value[string]{Present: true, Value: new("2026-02-03")},
			},
			want: reads.ErrConflictingTerminalDates,
		},
		{
			name: "finished_at is not before started_at",
			input: reads.UpdateReadRequest{
				StartedAt:  optional.Value[string]{Present: true, Value: new("2026-02-02")},
				FinishedAt: optional.Value[string]{Present: true, Value: new("2026-02-01")},
			},
			want: reads.ErrFinishedBeforeStarted,
		},
		{
			name: "abandoned_at is not before started_at",
			input: reads.UpdateReadRequest{
				StartedAt:   optional.Value[string]{Present: true, Value: new("2026-02-02")},
				AbandonedAt: optional.Value[string]{Present: true, Value: new("2026-02-01")},
			},
			want: reads.ErrAbandonedBeforeStarted,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := testsupport.OpenMigratedDB(t)
			bookID := testsupport.InsertBookRow(t, db, "Dune", nil)
			const readID = int64(100)
			const timestamp = "2026-01-01T00:00:00Z"
			testsupport.InsertReadRow(t, db, testsupport.ReadRow{
				ID: readID, BookID: bookID, CreatedAt: timestamp,
			})

			_, err := reads.NewSQLiteRepository(db).Update(context.Background(), readID, tt.input)
			if !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}

			testsupport.AssertReadRow(t, db, readID, testsupport.ReadRowAssertion{BookID: bookID})
			var updatedAt string
			if err := db.QueryRow(`SELECT updated_at FROM reads WHERE id = ?`, readID).Scan(&updatedAt); err != nil {
				t.Fatal(err)
			}
			if updatedAt != timestamp {
				t.Fatalf("failed update changed updated_at: got %q, want %q", updatedAt, timestamp)
			}
		})
	}
}

func TestSQLiteRepositoryUpdateLeavesUnknownCheckConstraintUnexpected(t *testing.T) {
	db, err := database.Open(context.Background(), filepath.Join(t.TempDir(), "unknown-constraint.db"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	_, err = db.Exec(`
		CREATE TABLE reads (
			id INTEGER PRIMARY KEY,
			book_id INTEGER NOT NULL,
			started_at TEXT NULL,
			finished_at TEXT NULL,
			abandoned_at TEXT NULL,
			rating REAL NULL,
			notes TEXT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			CONSTRAINT reads_unrecognized CHECK (notes <> 'blocked')
		);
		INSERT INTO reads (id, book_id, notes, created_at, updated_at)
		VALUES (100, 1, 'original', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z');
	`)
	if err != nil {
		t.Fatal(err)
	}

	_, err = reads.NewSQLiteRepository(db).Update(context.Background(), 100, reads.UpdateReadRequest{
		Notes: optional.Value[string]{Present: true, Value: new("blocked")},
	})
	if err == nil || !strings.Contains(err.Error(), "reads_unrecognized") {
		t.Fatalf("expected raw unknown constraint error, got %v", err)
	}
	assertNotReadValidationError(t, err)
}

func TestSQLiteRepositoryUpdateLeavesUnrelatedDatabaseErrorUnexpected(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	repository := reads.NewSQLiteRepository(db)
	if err := db.Close(); err != nil {
		t.Fatal(err)
	}

	_, err := repository.Update(context.Background(), 100, reads.UpdateReadRequest{
		Notes: optional.Value[string]{Present: true, Value: new("changed")},
	})
	if err == nil {
		t.Fatal("expected closed database error")
	}
	assertNotReadValidationError(t, err)
}

func assertNotReadValidationError(t *testing.T, err error) {
	t.Helper()

	validationErrors := []error{
		reads.ErrInvalidStartedAt,
		reads.ErrInvalidFinishedAt,
		reads.ErrInvalidAbandonedAt,
		reads.ErrInvalidRating,
		reads.ErrConflictingTerminalDates,
		reads.ErrFinishedBeforeStarted,
		reads.ErrAbandonedBeforeStarted,
	}
	for _, validationErr := range validationErrors {
		if errors.Is(err, validationErr) {
			t.Fatalf("unexpectedly translated %v to %v", err, validationErr)
		}
	}
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestSQLiteRepositoryDeleteReturnsReadNotFound(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)

	err := reads.NewSQLiteRepository(db).Delete(context.Background(), 999999)
	if !errors.Is(err, reads.ErrReadNotFound) {
		t.Fatalf("expected ErrReadNotFound, got %v", err)
	}
}

func TestSQLiteRepositoryDeletePreservesSiblingReadAndBooks(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)
	otherBookID := testsupport.InsertBookRow(t, db, "Foundation", nil)
	deletedReadID := int64(100)
	siblingReadID := int64(101)
	testsupport.InsertReadRow(t, db, testsupport.ReadRow{
		ID: deletedReadID, BookID: bookID, StartedAt: new("2026-01-01"),
		CreatedAt: "2026-01-01T00:00:00Z",
	})
	testsupport.InsertReadRow(t, db, testsupport.ReadRow{
		ID: siblingReadID, BookID: bookID, StartedAt: new("2026-02-01"),
		CreatedAt: "2026-02-01T00:00:00Z",
	})

	if err := reads.NewSQLiteRepository(db).Delete(context.Background(), deletedReadID); err != nil {
		t.Fatal(err)
	}

	var siblingCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM reads WHERE id = ?`, siblingReadID).Scan(&siblingCount); err != nil {
		t.Fatal(err)
	}
	if siblingCount != 1 {
		t.Fatalf("expected sibling read to remain, got %d rows", siblingCount)
	}

	var bookCount int
	if err := db.QueryRow(`SELECT COUNT(*) FROM books WHERE id IN (?, ?)`, bookID, otherBookID).Scan(&bookCount); err != nil {
		t.Fatal(err)
	}
	if bookCount != 2 {
		t.Fatalf("expected both books to remain, got %d rows", bookCount)
	}
}
