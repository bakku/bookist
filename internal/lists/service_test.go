package lists_test

import (
	"context"
	"errors"
	"testing"

	"bakku.dev/bookist/internal/lists"
	"bakku.dev/bookist/internal/optional"
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

func TestServiceCreateRejectsCaseInsensitiveDuplicate(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	service := lists.NewService(lists.NewSQLiteRepository(db))

	if _, err := service.Create(context.Background(), lists.CreateListRequest{Name: "Nightstand"}); err != nil {
		t.Fatal(err)
	}
	if _, err := service.Create(context.Background(), lists.CreateListRequest{Name: "NIGHTSTAND"}); !errors.Is(err, lists.ErrNameConflict) {
		t.Fatalf("expected ErrNameConflict, got %v", err)
	}
	testsupport.AssertListCount(t, db, 1)
}

// ── Update ────────────────────────────────────────────────────────────────────

func TestServiceUpdateAppliesPartialTrimmedFields(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	service := lists.NewService(lists.NewSQLiteRepository(db))
	id := testsupport.InsertListRowWithDescription(t, db, "Nightstand", "Old description")
	name := "  Bedside  "

	updated, err := service.Update(context.Background(), id, lists.UpdateListRequest{
		Name: optional.Value[string]{Present: true, Value: &name},
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Bedside" || updated.Description == nil || *updated.Description != "Old description" {
		t.Fatalf("unexpected updated list: %#v", updated)
	}
}

func TestServiceUpdateDescriptionRequiresExplicitNullToClear(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	service := lists.NewService(lists.NewSQLiteRepository(db))
	id := testsupport.InsertListRowWithDescription(t, db, "Nightstand", "Up next")
	blank := "  "

	_, err := service.Update(context.Background(), id, lists.UpdateListRequest{
		Description: optional.Value[string]{Present: true, Value: &blank},
	})
	if !errors.Is(err, lists.ErrDescriptionRequired) {
		t.Fatalf("expected ErrDescriptionRequired, got %v", err)
	}

	updated, err := service.Update(context.Background(), id, lists.UpdateListRequest{
		Description: optional.Value[string]{Present: true},
	})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Description != nil {
		t.Fatalf("expected description to be cleared, got %q", *updated.Description)
	}
}

func TestServiceUpdateValidatesRequestAndNameUniqueness(t *testing.T) {
	tests := []struct {
		name  string
		input lists.UpdateListRequest
		want  error
	}{
		{name: "empty request", want: lists.ErrNoFieldsToUpdate},
		{name: "null name", input: lists.UpdateListRequest{Name: optional.Value[string]{Present: true}}, want: lists.ErrNameRequired},
		{name: "blank name", input: lists.UpdateListRequest{Name: optional.Value[string]{Present: true, Value: listStringPointer(" ")}}, want: lists.ErrNameRequired},
		{name: "duplicate name", input: lists.UpdateListRequest{Name: optional.Value[string]{Present: true, Value: listStringPointer("NIGHTSTAND")}}, want: lists.ErrNameConflict},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			db := testsupport.OpenMigratedDB(t)
			service := lists.NewService(lists.NewSQLiteRepository(db))
			testsupport.InsertListRow(t, db, "Nightstand")
			id := testsupport.InsertListRow(t, db, "Want to Buy")

			_, err := service.Update(context.Background(), id, tt.input)
			if !errors.Is(err, tt.want) {
				t.Fatalf("expected %v, got %v", tt.want, err)
			}
			testsupport.AssertListRow(t, db, id, "Want to Buy")
		})
	}
}

func TestServiceUpdateAllowsCurrentListName(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	service := lists.NewService(lists.NewSQLiteRepository(db))
	id := testsupport.InsertListRow(t, db, "Nightstand")
	name := "NIGHTSTAND"

	if _, err := service.Update(context.Background(), id, lists.UpdateListRequest{
		Name: optional.Value[string]{Present: true, Value: &name},
	}); err != nil {
		t.Fatal(err)
	}
}

func TestServiceUpdateReturnsNotFoundBeforeNameConflict(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	service := lists.NewService(lists.NewSQLiteRepository(db))
	testsupport.InsertListRow(t, db, "Nightstand")
	name := "NIGHTSTAND"

	_, err := service.Update(context.Background(), 999999, lists.UpdateListRequest{
		Name: optional.Value[string]{Present: true, Value: &name},
	})
	if !errors.Is(err, lists.ErrListNotFound) {
		t.Fatalf("expected ErrListNotFound, got %v", err)
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
	testsupport.AssertListCount(t, db, 0)
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

func listStringPointer(value string) *string {
	return &value
}
