package authors_test

import (
	"context"
	"testing"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/testsupport"
)

// ── Create ────────────────────────────────────────────────────────────────────

func TestSQLiteRepositoryCreatePersistsAuthor(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := authors.NewSQLiteRepository(db)

	created, err := repository.Create(ctx, authors.CreateAuthorRequest{Name: "Jane Austen"})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID <= 0 {
		t.Fatal("expected created author to have an ID")
	}

	testsupport.AssertAuthorRow(t, db, created.ID, "Jane Austen")
}

// ── List ──────────────────────────────────────────────────────────────────────

func TestSQLiteRepositoryListReadsPersistedAuthors(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := authors.NewSQLiteRepository(db)
	id := testsupport.InsertAuthorRow(t, db, "Jane Austen")

	listed, err := repository.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected 1 author, got %d", len(listed))
	}
	if listed[0].ID != id {
		t.Fatalf("expected listed ID %d, got %d", id, listed[0].ID)
	}
	if listed[0].Name != "Jane Austen" {
		t.Fatalf("expected Jane Austen, got %q", listed[0].Name)
	}
}

// ── GetByIDs ──────────────────────────────────────────────────────────────────

func TestSQLiteRepositoryGetByIDsEmptyInputReturnsNil(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := authors.NewSQLiteRepository(db)

	found, err := repository.GetByIDs(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if found != nil {
		t.Fatalf("expected nil, got %d", len(found))
	}
}

func TestSQLiteRepositoryGetByIDsReturnsMatchingAuthors(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := authors.NewSQLiteRepository(db)
	id1 := testsupport.InsertAuthorRow(t, db, "Author One")
	id2 := testsupport.InsertAuthorRow(t, db, "Author Two")

	found, err := repository.GetByIDs(ctx, []int64{id1, id2, 999999})
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 2 {
		t.Fatalf("expected 2 authors, got %d", len(found))
	}
}

// ── ListByBookIDs ──────────────────────────────────────────────────────────────

func TestSQLiteRepositoryListByBookIDsEmptyInputReturnsEmptyMap(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := authors.NewSQLiteRepository(db)

	result, err := repository.ListByBookIDs(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	if result == nil {
		t.Fatal("expected non-nil map")
	}
	if len(result) != 0 {
		t.Fatalf("expected empty map, got %d entries", len(result))
	}
}

func TestSQLiteRepositoryListByBookIDsReturnsAuthorsGroupedByBook(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := authors.NewSQLiteRepository(db)

	bookID1 := testsupport.InsertBookRow(t, db, "Book One", nil)
	bookID2 := testsupport.InsertBookRow(t, db, "Book Two", nil)
	authorID := testsupport.InsertAuthorRow(t, db, "Shared Author")
	testsupport.InsertBookAuthorRow(t, db, bookID1, authorID)
	testsupport.InsertBookAuthorRow(t, db, bookID2, authorID)

	result, err := repository.ListByBookIDs(ctx, []int64{bookID1, bookID2})
	if err != nil {
		t.Fatal(err)
	}

	if len(result[bookID1]) != 1 {
		t.Fatalf("expected 1 author for book1, got %d", len(result[bookID1]))
	}
	if result[bookID1][0].Name != "Shared Author" {
		t.Fatalf("expected Shared Author, got %q", result[bookID1][0].Name)
	}
	if len(result[bookID2]) != 1 {
		t.Fatalf("expected 1 author for book2, got %d", len(result[bookID2]))
	}
}
