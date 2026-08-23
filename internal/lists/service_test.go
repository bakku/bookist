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
	testsupport.AssertSQLCount(t, db, 0, `SELECT COUNT(*) FROM lists`)
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
	testsupport.AssertListRow(t, db, created.ID, "Want to Buy", nil)
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

func TestServiceCreateRejectsCaseInsensitiveDuplicate(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	service := lists.NewService(lists.NewSQLiteRepository(db))

	if _, err := service.Create(context.Background(), lists.CreateListRequest{Name: "Nightstand"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Create(context.Background(), lists.CreateListRequest{Name: "NIGHTSTAND"}); !errors.Is(err, lists.ErrNameConflict) {
		t.Fatalf("expected ErrNameConflict, got %v", err)
	}
	testsupport.AssertSQLCount(t, db, 1, `SELECT COUNT(*) FROM lists`)
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

// ── Search ────────────────────────────────────────────────────────────────────

func TestServiceSearchTrimsQuery(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	service := lists.NewService(lists.NewSQLiteRepository(db))
	testsupport.InsertListRow(t, db, "Nightstand")
	testsupport.InsertListRow(t, db, "Want to Buy")

	matched, err := service.Search(context.Background(), "  NIGHT  ")
	if err != nil {
		t.Fatal(err)
	}
	if len(matched) != 1 || matched[0].Name != "Nightstand" {
		t.Fatalf("expected only Nightstand, got %#v", matched)
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

// ── Delete ────────────────────────────────────────────────────────────────────

func TestServiceDeleteDelegates(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	service := lists.NewService(lists.NewSQLiteRepository(db))
	id := testsupport.InsertListRow(t, db, "Want to Buy")

	if err := service.Delete(context.Background(), id); err != nil {
		t.Fatal(err)
	}
	testsupport.AssertSQLCount(t, db, 0, `SELECT COUNT(*) FROM lists`)
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

// ── RemoveBookFromList ────────────────────────────────────────────────────────

func TestServiceRemoveBookFromListDelegates(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	service := lists.NewService(lists.NewSQLiteRepository(db))
	listID := testsupport.InsertListRow(t, db, "Want to Buy")
	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)

	if err := service.AddBookToList(context.Background(), listID, bookID); err != nil {
		t.Fatal(err)
	}
	if err := service.RemoveBookFromList(context.Background(), listID, bookID); err != nil {
		t.Fatal(err)
	}

	var count int
	if err := db.QueryRow(`SELECT count(*) FROM book_lists WHERE list_id = ? AND book_id = ?`, listID, bookID).Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 0 {
		t.Fatalf("expected membership to be removed, got %d rows", count)
	}
}
