package lists_test

import (
	"context"
	"database/sql"
	"errors"
	"testing"
	"time"

	"bakku.dev/bookist/internal/domain"
	"bakku.dev/bookist/internal/lists"
	"bakku.dev/bookist/internal/optional"
	"bakku.dev/bookist/internal/testsupport"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// ── Create ────────────────────────────────────────────────────────────────────

func TestSQLiteRepositoryCreatePersistsList(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := lists.NewSQLiteRepository(db)

	created, err := repository.Create(ctx, lists.CreateListRequest{Name: "Want to Buy"})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID <= 0 {
		t.Fatal("expected created list to have an ID")
	}

	testsupport.AssertListRow(t, db, created.ID, "Want to Buy", nil)
}

func TestSQLiteRepositoryCreateWithDescription(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := lists.NewSQLiteRepository(db)

	desc := "Books I want to purchase"
	created, err := repository.Create(ctx, lists.CreateListRequest{Name: "Want to Buy", Description: &desc})
	if err != nil {
		t.Fatal(err)
	}
	if created.Description == nil || *created.Description != "Books I want to purchase" {
		t.Fatalf("expected description 'Books I want to purchase', got %#v", created.Description)
	}
}

func TestSQLiteRepositoryCreateWithNilDescription(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := lists.NewSQLiteRepository(db)

	created, err := repository.Create(ctx, lists.CreateListRequest{Name: "Want to Buy"})
	if err != nil {
		t.Fatal(err)
	}
	if created.Description != nil {
		t.Fatalf("expected nil description, got %q", *created.Description)
	}
}

func TestSQLiteRepositoryCreateReturnsErrNameConflict(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := lists.NewSQLiteRepository(db)
	testsupport.InsertListRow(t, db, "Favorites")

	_, err := repository.Create(ctx, lists.CreateListRequest{Name: "favorites"})
	if !errors.Is(err, lists.ErrNameConflict) {
		t.Fatalf("expected ErrNameConflict, got %v", err)
	}

	testsupport.AssertSQLCount(t, db, 1, `SELECT COUNT(*) FROM lists`)
}

func TestSQLiteRepositoryCreateLeavesConstraintLookalikeUnexpected(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := lists.NewSQLiteRepository(db)
	createAbortTrigger(t, db, "INSERT ON lists", "UNIQUE constraint failed")

	_, err := repository.Create(ctx, lists.CreateListRequest{Name: "Favorites"})
	assertTriggerError(t, err)
}

// ── Update ────────────────────────────────────────────────────────────────────

func TestSQLiteRepositoryUpdateClearsDescriptionAndPreservesOtherFields(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := lists.NewSQLiteRepository(db)
	id := testsupport.InsertListRowWithDescription(t, db, "Nightstand", "Up next")

	updated, err := repository.Update(ctx, id, lists.UpdateListRequest{
		Description: optional.Value[string]{Present: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.ID != id || updated.Name != "Nightstand" || updated.Description != nil {
		t.Fatalf("unexpected updated list: %#v", updated)
	}
	if got := updated.CreatedAt.Format("2006-01-02T15:04:05Z07:00"); got != "2026-01-02T03:04:05Z" {
		t.Fatalf("expected created_at to be preserved, got %q", got)
	}
	if !updated.UpdatedAt.After(updated.CreatedAt) {
		t.Fatalf("expected updated_at after created_at, got %v", updated.UpdatedAt)
	}
}

func TestSQLiteRepositoryUpdateMapsErrors(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := lists.NewSQLiteRepository(db)
	testsupport.InsertListRow(t, db, "Nightstand")
	id := testsupport.InsertListRow(t, db, "Want to Buy")
	duplicate := "NIGHTSTAND"

	_, err := repository.Update(ctx, id, lists.UpdateListRequest{
		Name: optional.Value[string]{Present: true, Value: &duplicate},
	})
	if !errors.Is(err, lists.ErrNameConflict) {
		t.Fatalf("expected ErrNameConflict, got %v", err)
	}

	_, err = repository.Update(ctx, 999999, lists.UpdateListRequest{
		Description: optional.Value[string]{Present: true},
	})
	if !errors.Is(err, domain.ErrListNotFound) {
		t.Fatalf("expected ErrListNotFound, got %v", err)
	}
}

func TestSQLiteRepositoryUpdateLeavesConstraintLookalikeUnexpected(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := lists.NewSQLiteRepository(db)
	id := testsupport.InsertListRow(t, db, "Favorites")
	createAbortTrigger(t, db, "UPDATE ON lists", "UNIQUE constraint failed")

	_, err := repository.Update(ctx, id, lists.UpdateListRequest{
		Name: optional.Value[string]{Present: true, Value: new("Changed")},
	})
	assertTriggerError(t, err)
}

// ── List ──────────────────────────────────────────────────────────────────────

func TestSQLiteRepositoryListReadsPersistedLists(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := lists.NewSQLiteRepository(db)
	id1 := testsupport.InsertListRow(t, db, "Nightstand")
	id2 := testsupport.InsertListRow(t, db, "Want to Buy")

	listed, err := repository.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 2 {
		t.Fatalf("expected 2 lists, got %d", len(listed))
	}
	wantFirst, wantSecond := id1, id2
	if id2 < id1 {
		wantFirst, wantSecond = id2, id1
	}
	if listed[0].ID != wantFirst || listed[1].ID != wantSecond {
		t.Fatalf("expected ID tie-break ordering [%d %d], got [%d %d]", wantFirst, wantSecond, listed[0].ID, listed[1].ID)
	}
}

// ── Search ────────────────────────────────────────────────────────────────────

func TestSQLiteRepositorySearchMatchesNamesCaseInsensitively(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := lists.NewSQLiteRepository(db)
	id1 := testsupport.InsertListRow(t, db, "Bedroom Shelf")
	id2 := testsupport.InsertListRow(t, db, "Office Shelf")
	id3 := testsupport.InsertListRow(t, db, "Living Room Shelf")
	testsupport.InsertListRow(t, db, "Want to Buy")
	if _, err := db.Exec(`UPDATE lists SET updated_at = '2026-01-03T00:00:00Z' WHERE id = ?`, id3); err != nil {
		t.Fatal(err)
	}

	matched, err := repository.Search(ctx, "sHeLf")
	if err != nil {
		t.Fatal(err)
	}
	if len(matched) != 3 {
		t.Fatalf("expected 3 matching lists, got %#v", matched)
	}
	if matched[0].ID != id3 || matched[1].ID != id1 || matched[2].ID != id2 {
		t.Fatalf("expected updated-at and ID ordering [%d %d %d], got [%d %d %d]", id3, id1, id2, matched[0].ID, matched[1].ID, matched[2].ID)
	}
}

// ── GetByID ───────────────────────────────────────────────────────────────────

func TestSQLiteRepositoryGetByIDReturnsList(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := lists.NewSQLiteRepository(db)
	id := testsupport.InsertListRow(t, db, "Want to Buy")

	got, err := repository.GetByID(ctx, id)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != id {
		t.Fatalf("expected ID %d, got %d", id, got.ID)
	}
	if got.Name != "Want to Buy" {
		t.Fatalf("expected Want to Buy, got %q", got.Name)
	}
}

func TestSQLiteRepositoryGetByIDReturnsErrListNotFound(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := lists.NewSQLiteRepository(db)

	_, err := repository.GetByID(ctx, 999999)
	if !errors.Is(err, domain.ErrListNotFound) {
		t.Fatalf("expected ErrListNotFound, got %v", err)
	}
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestSQLiteRepositoryDeleteRemovesListAndCascadesMemberships(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := lists.NewSQLiteRepository(db)
	listID := testsupport.InsertListRow(t, db, "Want to Buy")
	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)
	if err := repository.AddBookToList(ctx, listID, bookID); err != nil {
		t.Fatal(err)
	}

	if err := repository.Delete(ctx, listID); err != nil {
		t.Fatal(err)
	}

	var listCount, membershipCount, bookCount int
	if err := db.QueryRow(`SELECT count(*) FROM lists WHERE id = ?`, listID).Scan(&listCount); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM book_lists WHERE list_id = ?`, listID).Scan(&membershipCount); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM books WHERE id = ?`, bookID).Scan(&bookCount); err != nil {
		t.Fatal(err)
	}
	if listCount != 0 || membershipCount != 0 || bookCount != 1 {
		t.Fatalf("expected list and membership deleted but book preserved, got list=%d membership=%d book=%d", listCount, membershipCount, bookCount)
	}
}

func TestSQLiteRepositoryDeleteReturnsErrListNotFound(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	repository := lists.NewSQLiteRepository(db)

	err := repository.Delete(context.Background(), 999999)
	if !errors.Is(err, domain.ErrListNotFound) {
		t.Fatalf("expected ErrListNotFound, got %v", err)
	}
}

// ── AddBookToList ─────────────────────────────────────────────────────────────

func TestSQLiteRepositoryAddBookToListPersistsRow(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := lists.NewSQLiteRepository(db)

	listID := testsupport.InsertListRow(t, db, "Want to Buy")
	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)

	err := repository.AddBookToList(ctx, listID, bookID)
	if err != nil {
		t.Fatal(err)
	}

	testsupport.AssertBookListRow(t, db, listID, bookID)
}

func TestSQLiteRepositoryAddBookToListReturnsErrBookAlreadyInList(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := lists.NewSQLiteRepository(db)

	listID := testsupport.InsertListRow(t, db, "Want to Buy")
	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)

	err := repository.AddBookToList(ctx, listID, bookID)
	if err != nil {
		t.Fatal(err)
	}

	err = repository.AddBookToList(ctx, listID, bookID)
	if !errors.Is(err, lists.ErrBookAlreadyInList) {
		t.Fatalf("expected ErrBookAlreadyInList, got %v", err)
	}
}

func TestSQLiteRepositoryAddBookToListReturnsErrListNotFound(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := lists.NewSQLiteRepository(db)

	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)

	err := repository.AddBookToList(ctx, 999999, bookID)
	if !errors.Is(err, domain.ErrListNotFound) {
		t.Fatalf("expected ErrListNotFound, got %v", err)
	}
}

func TestSQLiteRepositoryAddBookToListReturnsErrBookNotFound(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := lists.NewSQLiteRepository(db)

	listID := testsupport.InsertListRow(t, db, "Want to Buy")

	err := repository.AddBookToList(ctx, listID, 999999)
	if !errors.Is(err, domain.ErrBookNotFound) {
		t.Fatalf("expected ErrBookNotFound, got %v", err)
	}
}

func TestSQLiteRepositoryAddBookToListReturnsListLookupParseErrorUnchanged(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := lists.NewSQLiteRepository(db)

	listID := testsupport.InsertListRow(t, db, "Favorites")
	if _, err := db.Exec(`UPDATE lists SET created_at = 'invalid' WHERE id = ?`, listID); err != nil {
		t.Fatal(err)
	}

	err := repository.AddBookToList(ctx, listID, 999999)
	var parseErr *time.ParseError
	if !errors.As(err, &parseErr) {
		t.Fatalf("expected lookup parse error, got %v", err)
	}

	testsupport.AssertSQLCount(t, db, 0, `SELECT COUNT(*) FROM book_lists`)
}

func TestSQLiteRepositoryAddBookToListReturnsListLookupDatabaseErrorUnchanged(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := lists.NewSQLiteRepository(db)

	listID := testsupport.InsertListRow(t, db, "Favorites")
	if _, err := db.Exec(`ALTER TABLE lists RENAME COLUMN description TO unavailable_description`); err != nil {
		t.Fatal(err)
	}

	err := repository.AddBookToList(ctx, listID, 999999)
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) || sqliteErr.Code() != sqlite3.SQLITE_ERROR {
		t.Fatalf("expected lookup database error, got %v", err)
	}

	testsupport.AssertSQLCount(t, db, 0, `SELECT COUNT(*) FROM book_lists`)
}

func TestSQLiteRepositoryAddBookToListLeavesConstraintLookalikesUnexpected(t *testing.T) {
	for _, message := range []string{"UNIQUE constraint failed", "FOREIGN KEY constraint failed"} {
		t.Run(message, func(t *testing.T) {
			ctx := context.Background()
			db := testsupport.OpenMigratedDB(t)
			repository := lists.NewSQLiteRepository(db)

			listID := testsupport.InsertListRow(t, db, "Favorites")
			bookID := testsupport.InsertBookRow(t, db, "Dune", nil)
			createAbortTrigger(t, db, "INSERT ON book_lists", message)

			err := repository.AddBookToList(ctx, listID, bookID)
			assertTriggerError(t, err)
		})
	}
}

// ── RemoveBookFromList ────────────────────────────────────────────────────────

func TestSQLiteRepositoryRemoveBookFromListRemovesExactMembershipOnly(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := lists.NewSQLiteRepository(db)
	listID := testsupport.InsertListRow(t, db, "Want to Buy")
	otherListID := testsupport.InsertListRow(t, db, "Nightstand")
	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)
	otherBookID := testsupport.InsertBookRow(t, db, "Neuromancer", nil)

	for _, membership := range [][2]int64{{listID, bookID}, {listID, otherBookID}, {otherListID, bookID}} {
		if err := repository.AddBookToList(ctx, membership[0], membership[1]); err != nil {
			t.Fatal(err)
		}
	}

	if err := repository.RemoveBookFromList(ctx, listID, bookID); err != nil {
		t.Fatal(err)
	}

	var removedCount, unrelatedCount int
	if err := db.QueryRow(`SELECT count(*) FROM book_lists WHERE list_id = ? AND book_id = ?`, listID, bookID).Scan(&removedCount); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT count(*) FROM book_lists`).Scan(&unrelatedCount); err != nil {
		t.Fatal(err)
	}
	if removedCount != 0 || unrelatedCount != 2 {
		t.Fatalf("expected exact membership removed and 2 unrelated rows preserved, got removed=%d total=%d", removedCount, unrelatedCount)
	}
}

func TestSQLiteRepositoryRemoveBookFromListClassifiesMissingRows(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(t *testing.T, db *sql.DB) (int64, int64)
		wantError error
	}{
		{
			name: "list takes precedence when both are missing",
			setup: func(t *testing.T, db *sql.DB) (int64, int64) {
				return 999998, 999999
			},
			wantError: domain.ErrListNotFound,
		},
		{
			name: "book missing",
			setup: func(t *testing.T, db *sql.DB) (int64, int64) {
				return testsupport.InsertListRow(t, db, "Want to Buy"), 999999
			},
			wantError: domain.ErrBookNotFound,
		},
		{
			name: "membership missing",
			setup: func(t *testing.T, db *sql.DB) (int64, int64) {
				return testsupport.InsertListRow(t, db, "Want to Buy"), testsupport.InsertBookRow(t, db, "Dune", nil)
			},
			wantError: lists.ErrBookNotInList,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := testsupport.OpenMigratedDB(t)
			repository := lists.NewSQLiteRepository(db)
			listID, bookID := tt.setup(t, db)

			err := repository.RemoveBookFromList(context.Background(), listID, bookID)
			if !errors.Is(err, tt.wantError) {
				t.Fatalf("expected %v, got %v", tt.wantError, err)
			}
		})
	}
}

// createAbortTrigger makes the given statement fail with constraint-like text
// but a trigger error code, so translations must not rely on the message.
func createAbortTrigger(t *testing.T, db *sql.DB, event, message string) {
	t.Helper()

	if _, err := db.Exec(`CREATE TRIGGER blocked BEFORE ` + event + ` BEGIN SELECT RAISE(ABORT, '` + message + `'); END`); err != nil {
		t.Fatal(err)
	}
}

func assertTriggerError(t *testing.T, err error) {
	t.Helper()

	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) || sqliteErr.Code() != sqlite3.SQLITE_CONSTRAINT_TRIGGER {
		t.Fatalf("expected unchanged trigger error, got %v", err)
	}
}
