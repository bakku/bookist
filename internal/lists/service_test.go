package lists_test

import (
	"context"
	"errors"
	"testing"

	"bakku.dev/bookist/internal/lists"
	"bakku.dev/bookist/internal/testsupport"
)

// ── Create ────────────────────────────────────────────────────────────────────

func TestServiceCreateRequiresName(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	service := lists.NewService(lists.NewSQLiteRepository(db))

	_, err := service.Create(context.Background(), lists.CreateListRequest{Name: " "})
	if !errors.Is(err, lists.ErrNameRequired) {
		t.Fatalf("expected ErrNameRequired, got %v", err)
	}
	testsupport.AssertListCount(t, db, 0)
}

func TestServiceCreateTrimsAndPersistsName(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	service := lists.NewService(lists.NewSQLiteRepository(db))

	created, err := service.Create(context.Background(), lists.CreateListRequest{Name: "  Want to Buy  "})
	if err != nil {
		t.Fatal(err)
	}
	if created.Name != "Want to Buy" {
		t.Fatalf("expected trimmed name, got %q", created.Name)
	}
	testsupport.AssertListRow(t, db, created.ID, "Want to Buy")
}

func TestServiceCreateTrimsAndPersistsDescription(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	service := lists.NewService(lists.NewSQLiteRepository(db))

	desc := "  Books I want  "
	created, err := service.Create(context.Background(), lists.CreateListRequest{Name: "Want to Buy", Description: &desc})
	if err != nil {
		t.Fatal(err)
	}
	if created.Description == nil || *created.Description != "Books I want" {
		t.Fatalf("expected trimmed description 'Books I want', got %#v", created.Description)
	}
}

func TestServiceCreateConvertsBlankDescriptionToNil(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	service := lists.NewService(lists.NewSQLiteRepository(db))

	desc := "  "
	created, err := service.Create(context.Background(), lists.CreateListRequest{Name: "Want to Buy", Description: &desc})
	if err != nil {
		t.Fatal(err)
	}
	if created.Description != nil {
		t.Fatalf("expected nil description, got %q", *created.Description)
	}
}

// ── List ──────────────────────────────────────────────────────────────────────

func TestServiceListDelegates(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	service := lists.NewService(lists.NewSQLiteRepository(db))
	testsupport.InsertListRow(t, db, "Want to Buy")

	listed, err := service.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected 1 list, got %d", len(listed))
	}
	if listed[0].Name != "Want to Buy" {
		t.Fatalf("expected Want to Buy, got %q", listed[0].Name)
	}
}

// ── GetByID ───────────────────────────────────────────────────────────────────

func TestServiceGetByIDReturnsList(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	service := lists.NewService(lists.NewSQLiteRepository(db))
	id := testsupport.InsertListRow(t, db, "Want to Buy")

	got, err := service.GetByID(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	if got.Name != "Want to Buy" {
		t.Fatalf("expected Want to Buy, got %q", got.Name)
	}
}

// ── AddBookToList ─────────────────────────────────────────────────────────────

func TestServiceAddBookToListDelegates(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	service := lists.NewService(lists.NewSQLiteRepository(db))

	listID := testsupport.InsertListRow(t, db, "Want to Buy")
	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)

	err := service.AddBookToList(context.Background(), listID, bookID)
	if err != nil {
		t.Fatal(err)
	}

	testsupport.AssertBookListRow(t, db, listID, bookID)
}

// ── ListBooks ─────────────────────────────────────────────────────────────────

func TestServiceListBooksReturnsBooks(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	service := lists.NewService(lists.NewSQLiteRepository(db))

	listID := testsupport.InsertListRow(t, db, "Want to Buy")
	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)

	if err := service.AddBookToList(context.Background(), listID, bookID); err != nil {
		t.Fatal(err)
	}

	bookList, err := service.ListBooks(context.Background(), listID)
	if err != nil {
		t.Fatal(err)
	}
	if len(bookList) != 1 {
		t.Fatalf("expected 1 book, got %d", len(bookList))
	}
	if bookList[0].Title != "Dune" {
		t.Fatalf("expected Dune, got %q", bookList[0].Title)
	}
}
