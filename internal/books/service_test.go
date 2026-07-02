package books_test

import (
	"context"
	"errors"
	"testing"

	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/testsupport"
	"github.com/google/uuid"
)

// ── Create ────────────────────────────────────────────────────────────────────

func TestServiceCreateConvertsBlankISBNToPersistedNull(t *testing.T) {
	service, db := testsupport.NewBookService(t)
	isbn := " "

	created, err := service.Create(context.Background(), books.CreateBookRequest{Title: "Dune", ISBN: &isbn})
	if err != nil {
		t.Fatal(err)
	}
	if created.ISBN != nil {
		t.Fatalf("expected nil ISBN, got %#v", created.ISBN)
	}
	testsupport.AssertBookRow(t, db, created.ID, "Dune", nil)
}

func TestServiceCreateRequiresTitle(t *testing.T) {
	service, db := testsupport.NewBookService(t)

	_, err := service.Create(context.Background(), books.CreateBookRequest{Title: " "})
	if !errors.Is(err, books.ErrTitleRequired) {
		t.Fatalf("expected ErrTitleRequired, got %v", err)
	}
	testsupport.AssertBookCount(t, db, 0)
}

func TestServiceCreateRejectsUnknownAuthorIDs(t *testing.T) {
	service, db := testsupport.NewBookService(t)

	_, err := service.Create(context.Background(), books.CreateBookRequest{
		Title:     "Test",
		AuthorIDs: []string{uuid.NewString()},
	})
	if !errors.Is(err, books.ErrAuthorNotFound) {
		t.Fatalf("expected ErrAuthorNotFound, got %v", err)
	}
	testsupport.AssertBookCount(t, db, 0)
}

func TestServiceCreateTrimsAndPersistsInput(t *testing.T) {
	service, db := testsupport.NewBookService(t)
	isbn := " 9783161484100 "

	created, err := service.Create(context.Background(), books.CreateBookRequest{
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

func TestServiceCreateWithNoAuthorsReturnsEmptyNonNilSlice(t *testing.T) {
	service, _ := testsupport.NewBookService(t)

	created, err := service.Create(context.Background(), books.CreateBookRequest{Title: "No Authors"})
	if err != nil {
		t.Fatal(err)
	}
	if created.Authors == nil {
		t.Fatal("expected non-nil Authors slice")
	}
	if len(created.Authors) != 0 {
		t.Fatalf("expected empty Authors slice, got %d", len(created.Authors))
	}
}

func TestServiceCreateWithValidAuthorIDsReturnsHydratedAuthors(t *testing.T) {
	service, db := testsupport.NewBookService(t)

	author := testsupport.InsertAuthorRow(t, db, "Test Author")
	created, err := service.Create(context.Background(), books.CreateBookRequest{
		Title:     "Test Book",
		AuthorIDs: []string{author},
	})
	if err != nil {
		t.Fatal(err)
	}
	if len(created.Authors) != 1 {
		t.Fatalf("expected 1 author, got %d", len(created.Authors))
	}
	if created.Authors[0].Name != "Test Author" {
		t.Fatalf("expected author name 'Test Author', got %q", created.Authors[0].Name)
	}
	testsupport.AssertBookAuthors(t, db, created.ID, author)
}

// ── List ──────────────────────────────────────────────────────────────────────

func TestServiceListHydratesAuthors(t *testing.T) {
	service, db := testsupport.NewBookService(t)

	bookID := testsupport.InsertBookRow(t, db, "Book With Author", nil)
	author := testsupport.InsertAuthorRow(t, db, "Author One")
	testsupport.InsertBookAuthorRow(t, db, bookID, author)

	listed, err := service.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected 1 book, got %d", len(listed))
	}
	if len(listed[0].Authors) != 1 {
		t.Fatalf("expected 1 author on listed book, got %d", len(listed[0].Authors))
	}
	if listed[0].Authors[0].Name != "Author One" {
		t.Fatalf("expected Author One, got %q", listed[0].Authors[0].Name)
	}
}

func TestServiceListReturnsPersistedBooks(t *testing.T) {
	service, db := testsupport.NewBookService(t)
	id := testsupport.InsertBookRow(t, db, "Kindred", nil)

	listed, err := service.List(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected 1 book, got %d", len(listed))
	}
	if listed[0].ID != id {
		t.Fatalf("expected listed ID %s, got %s", id, listed[0].ID)
	}
	if listed[0].Title != "Kindred" {
		t.Fatalf("expected Kindred, got %q", listed[0].Title)
	}
}
