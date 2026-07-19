package lists_test

import (
	"context"
	"errors"
	"testing"

	"bakku.dev/bookist/internal/lists"
	"bakku.dev/bookist/internal/testsupport"
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

	testsupport.AssertListRow(t, db, created.ID, "Want to Buy")
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
	if !errors.Is(err, lists.ErrListNotFound) {
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
	if !errors.Is(err, lists.ErrListNotFound) {
		t.Fatalf("expected ErrListNotFound, got %v", err)
	}
}

func TestSQLiteRepositoryAddBookToListReturnsErrBookNotFound(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := lists.NewSQLiteRepository(db)

	listID := testsupport.InsertListRow(t, db, "Want to Buy")

	err := repository.AddBookToList(ctx, listID, 999999)
	if !errors.Is(err, lists.ErrBookNotFound) {
		t.Fatalf("expected ErrBookNotFound, got %v", err)
	}
}
