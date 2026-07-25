package authors_test

import (
	"context"
	"errors"
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

// ── Search ────────────────────────────────────────────────────────────────────

func TestSQLiteRepositorySearchMatchesNamesCaseInsensitively(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := authors.NewSQLiteRepository(db)
	id1 := testsupport.InsertAuthorRow(t, db, "Jane Austen")
	id2 := testsupport.InsertAuthorRow(t, db, "Jane Goodall")
	id3 := testsupport.InsertAuthorRow(t, db, "Jane Yolen")
	testsupport.InsertAuthorRow(t, db, "Octavia Butler")
	if _, err := db.Exec(`UPDATE authors SET updated_at = '2026-01-03T00:00:00Z' WHERE id = ?`, id3); err != nil {
		t.Fatal(err)
	}

	matched, err := repository.Search(ctx, "jAnE")
	if err != nil {
		t.Fatal(err)
	}
	if len(matched) != 3 {
		t.Fatalf("expected 3 matching authors, got %#v", matched)
	}
	if matched[0].ID != id3 || matched[1].ID != id1 || matched[2].ID != id2 {
		t.Fatalf("expected updated-at and ID ordering [%d %d %d], got [%d %d %d]", id3, id1, id2, matched[0].ID, matched[1].ID, matched[2].ID)
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

// ── Delete ────────────────────────────────────────────────────────────────────

func TestSQLiteRepositoryDeletePersistsDeletion(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := authors.NewSQLiteRepository(db)
	id := testsupport.InsertAuthorRow(t, db, "Jane Austen")

	if err := repository.Delete(ctx, id); err != nil {
		t.Fatal(err)
	}

	testsupport.AssertAuthorCount(t, db, 0)
}

func TestSQLiteRepositoryDeleteReturnsErrAuthorNotFound(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	repository := authors.NewSQLiteRepository(db)

	err := repository.Delete(context.Background(), 999999)
	if !errors.Is(err, authors.ErrAuthorNotFound) {
		t.Fatalf("expected ErrAuthorNotFound, got %v", err)
	}
}

func TestSQLiteRepositoryDeleteCascadesRelationshipAndPreservesBook(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := authors.NewSQLiteRepository(db)
	bookID := testsupport.InsertBookRow(t, db, "Pride and Prejudice", nil)
	authorID := testsupport.InsertAuthorRow(t, db, "Jane Austen")
	testsupport.InsertBookAuthorRow(t, db, bookID, authorID)

	if err := repository.Delete(ctx, authorID); err != nil {
		t.Fatal(err)
	}

	testsupport.AssertBookHasNoAuthors(t, db, bookID)
	testsupport.AssertBookRow(t, db, bookID, "Pride and Prejudice", nil)
}
