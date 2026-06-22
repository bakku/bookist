package books_test

import (
	"context"
	"errors"
	"testing"

	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/testsupport"
)

func TestServiceCreateTrimsAndPersistsInput(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	service := books.NewService(books.NewSQLiteRepository(db))
	isbn := " 9783161484100 "

	created, err := service.Create(context.Background(), books.CreateBookInput{
		Title: " The Go Programming Language ",
		ISBN:  &isbn,
	})
	if err != nil {
		t.Fatal(err)
	}

	if created.Title != "The Go Programming Language" {
		t.Fatalf("expected trimmed title, got %q", created.Title)
	}
	if created.ISBN == nil || *created.ISBN != "9783161484100" {
		t.Fatalf("expected trimmed ISBN, got %#v", created.ISBN)
	}
	testsupport.AssertBookRow(t, db, created.ID, "The Go Programming Language", testsupport.StringPtr("9783161484100"))
}

func TestServiceCreateRequiresTitle(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	service := books.NewService(books.NewSQLiteRepository(db))

	_, err := service.Create(context.Background(), books.CreateBookInput{Title: " "})
	if !errors.Is(err, books.ErrTitleRequired) {
		t.Fatalf("expected ErrTitleRequired, got %v", err)
	}
	testsupport.AssertBookCount(t, db, 0)
}

func TestServiceCreateConvertsBlankISBNToPersistedNull(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	service := books.NewService(books.NewSQLiteRepository(db))
	isbn := " "

	created, err := service.Create(context.Background(), books.CreateBookInput{Title: "Dune", ISBN: &isbn})
	if err != nil {
		t.Fatal(err)
	}
	if created.ISBN != nil {
		t.Fatalf("expected nil ISBN, got %#v", created.ISBN)
	}
	testsupport.AssertBookRow(t, db, created.ID, "Dune", nil)
}

func TestServiceListReturnsPersistedBooks(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	service := books.NewService(books.NewSQLiteRepository(db))
	id := testsupport.InsertBookRow(t, db, "Kindred", nil)

	listed, err := service.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected 1 book, got %d", len(listed))
	}
	if listed[0].ID != id {
		t.Fatalf("expected listed ID %d, got %d", id, listed[0].ID)
	}
	if listed[0].Title != "Kindred" {
		t.Fatalf("expected Kindred, got %q", listed[0].Title)
	}
}
