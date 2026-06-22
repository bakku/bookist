package books_test

import (
	"context"
	"testing"

	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/testsupport"
)

func TestSQLiteRepositoryCreatePersistsBook(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := books.NewSQLiteRepository(db)
	isbn := "9783161484100"

	created, err := repository.Create(ctx, books.CreateBookInput{Title: "The Go Programming Language", ISBN: &isbn})
	if err != nil {
		t.Fatal(err)
	}
	if created.ID == 0 {
		t.Fatal("expected created book to have an ID")
	}

	testsupport.AssertBookRow(t, db, created.ID, "The Go Programming Language", &isbn)
}

func TestSQLiteRepositoryCreatePersistsNullISBN(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := books.NewSQLiteRepository(db)

	created, err := repository.Create(ctx, books.CreateBookInput{Title: "Kindred"})
	if err != nil {
		t.Fatal(err)
	}

	testsupport.AssertBookRow(t, db, created.ID, "Kindred", nil)
}

func TestSQLiteRepositoryListReadsPersistedBooks(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := books.NewSQLiteRepository(db)
	id := testsupport.InsertBookRow(t, db, "The Go Programming Language", testsupport.StringPtr("9783161484100"))

	listed, err := repository.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected 1 book, got %d", len(listed))
	}
	if listed[0].ID != id {
		t.Fatalf("expected listed ID %d, got %d", id, listed[0].ID)
	}
	if listed[0].Title != "The Go Programming Language" {
		t.Fatalf("expected listed title, got %q", listed[0].Title)
	}
	if listed[0].ISBN == nil || *listed[0].ISBN != "9783161484100" {
		t.Fatalf("expected listed ISBN, got %#v", listed[0].ISBN)
	}
}
