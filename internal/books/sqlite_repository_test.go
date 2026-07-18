package books_test

import (
	"context"
	"testing"

	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/testsupport"
)

// ── Create ────────────────────────────────────────────────────────────────────

func TestSQLiteRepositoryCreatePersistsAllFields(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := books.NewSQLiteRepository(db)

	isbn := "9783161484100"
	lang := "en"
	pub := "O'Reilly"
	ed := "2nd"
	format := books.FormatPaperback
	purchased := "2025-06-15"
	pages := 400
	notes := "Great book"
	year := 2024
	month := 6
	day := 15

	created, err := repository.Create(ctx, books.CreateBookRequest{
		Title:          "The Go Programming Language",
		ISBN:           &isbn,
		Language:       &lang,
		Publisher:      &pub,
		Edition:        &ed,
		Format:         &format,
		PurchasedAt:    &purchased,
		Pages:          &pages,
		Notes:          &notes,
		PublishedYear:  &year,
		PublishedMonth: &month,
		PublishedDay:   &day,
	})
	if err != nil {
		t.Fatal(err)
	}

	if created.ID == "" {
		t.Fatal("expected created book to have an ID")
	}

	f := string(format)
	testsupport.AssertBookRowFields(t, db, created.ID, testsupport.BookRowAssertion{
		Title:          "The Go Programming Language",
		ISBN:           &isbn,
		Language:       &lang,
		Publisher:      &pub,
		Edition:        &ed,
		Format:         &f,
		PurchasedAt:    &purchased,
		Pages:          &pages,
		Notes:          &notes,
		PublishedYear:  &year,
		PublishedMonth: &month,
		PublishedDay:   &day,
	})
}

func TestSQLiteRepositoryCreatePersistsNullDefaults(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := books.NewSQLiteRepository(db)

	created, err := repository.Create(ctx, books.CreateBookRequest{Title: "Minimal Book"})
	if err != nil {
		t.Fatal(err)
	}

	testsupport.AssertBookRowFields(t, db, created.ID, testsupport.BookRowAssertion{
		Title: "Minimal Book",
	})
}

func TestSQLiteRepositoryCreateWithAuthorIDsPersistsBookAuthors(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := books.NewSQLiteRepository(db)

	authorID := testsupport.InsertAuthorRow(t, db, "Test Author")

	created, err := repository.Create(ctx, books.CreateBookRequest{
		Title:     "Book With Authors",
		AuthorIDs: []string{authorID},
	})
	if err != nil {
		t.Fatal(err)
	}

	testsupport.AssertBookRow(t, db, created.ID, "Book With Authors", nil)
	testsupport.AssertBookAuthors(t, db, created.ID, authorID)
}

func TestSQLiteRepositoryCreateWithNoAuthorIDsPersistsBookOnly(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := books.NewSQLiteRepository(db)

	created, err := repository.Create(ctx, books.CreateBookRequest{Title: "Solo Book"})
	if err != nil {
		t.Fatal(err)
	}

	testsupport.AssertBookRow(t, db, created.ID, "Solo Book", nil)
	testsupport.AssertBookAuthors(t, db, created.ID)
}

// ── List ──────────────────────────────────────────────────────────────────────

func TestSQLiteRepositoryListReadsPersistedBooks(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := books.NewSQLiteRepository(db)

	isbn := "9783161484100"
	lang := "en"
	pub := "O'Reilly"
	ed := "2nd"
	format := books.FormatPaperback
	purchased := "2025-06-15"
	pages := 400
	notes := "Great book"
	year := 2024
	month := 6
	day := 15

	created, err := repository.Create(ctx, books.CreateBookRequest{
		Title:          "The Go Programming Language",
		ISBN:           &isbn,
		Language:       &lang,
		Publisher:      &pub,
		Edition:        &ed,
		Format:         &format,
		PurchasedAt:    &purchased,
		Pages:          &pages,
		Notes:          &notes,
		PublishedYear:  &year,
		PublishedMonth: &month,
		PublishedDay:   &day,
	})
	if err != nil {
		t.Fatal(err)
	}

	listed, err := repository.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected 1 book, got %d", len(listed))
	}

	got := listed[0]

	if got.ID != created.ID {
		t.Fatalf("expected ID %s, got %s", created.ID, got.ID)
	}

	if got.Title != "The Go Programming Language" {
		t.Fatalf("expected title %q, got %q", "The Go Programming Language", got.Title)
	}

	if got.ISBN == nil || *got.ISBN != isbn {
		t.Fatalf("expected ISBN %q, got %#v", isbn, got.ISBN)
	}

	if got.Language == nil || *got.Language != lang {
		t.Fatalf("expected Language %q, got %#v", lang, got.Language)
	}

	if got.Publisher == nil || *got.Publisher != pub {
		t.Fatalf("expected Publisher %q, got %#v", pub, got.Publisher)
	}

	if got.Edition == nil || *got.Edition != ed {
		t.Fatalf("expected Edition %q, got %#v", ed, got.Edition)
	}

	if got.Format == nil || *got.Format != format {
		t.Fatalf("expected Format %q, got %#v", format, got.Format)
	}

	if got.PurchasedAt == nil || *got.PurchasedAt != purchased {
		t.Fatalf("expected PurchasedAt %q, got %#v", purchased, got.PurchasedAt)
	}

	if got.Pages == nil || *got.Pages != pages {
		t.Fatalf("expected Pages %d, got %#v", pages, got.Pages)
	}

	if got.Notes == nil || *got.Notes != notes {
		t.Fatalf("expected Notes %q, got %#v", notes, got.Notes)
	}

	if got.PublishedYear == nil || *got.PublishedYear != year {
		t.Fatalf("expected PublishedYear %d, got %#v", year, got.PublishedYear)
	}

	if got.PublishedMonth == nil || *got.PublishedMonth != month {
		t.Fatalf("expected PublishedMonth %d, got %#v", month, got.PublishedMonth)
	}

	if got.PublishedDay == nil || *got.PublishedDay != day {
		t.Fatalf("expected PublishedDay %d, got %#v", day, got.PublishedDay)
	}
}

// ── ListByListID ──────────────────────────────────────────────────────────────

func TestSQLiteRepositoryListByListIDReturnsBooksInList(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := books.NewSQLiteRepository(db)

	listID := testsupport.InsertListRow(t, db, "Want to Buy")
	bookID1 := testsupport.InsertBookRow(t, db, "Dune", nil)
	bookID2 := testsupport.InsertBookRow(t, db, "Foundation", nil)
	testsupport.InsertBookListRow(t, db, listID, bookID1)
	testsupport.InsertBookListRow(t, db, listID, bookID2)

	bookList, err := repository.ListByListID(ctx, listID)
	if err != nil {
		t.Fatal(err)
	}
	if len(bookList) != 2 {
		t.Fatalf("expected 2 books, got %d", len(bookList))
	}
	if bookList[0].Title != "Dune" {
		t.Fatalf("expected Dune, got %q", bookList[0].Title)
	}
	if bookList[1].Title != "Foundation" {
		t.Fatalf("expected Foundation, got %q", bookList[1].Title)
	}
}

func TestSQLiteRepositoryListByListIDReturnsEmptySliceForEmptyList(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := books.NewSQLiteRepository(db)

	listID := testsupport.InsertListRow(t, db, "Want to Buy")

	bookList, err := repository.ListByListID(ctx, listID)
	if err != nil {
		t.Fatal(err)
	}
	if bookList == nil {
		t.Fatal("expected non-nil slice")
	}
	if len(bookList) != 0 {
		t.Fatalf("expected empty slice, got %d books", len(bookList))
	}
}

func TestSQLiteRepositoryListByListIDDoesNotReturnBooksFromOtherLists(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := books.NewSQLiteRepository(db)

	listID1 := testsupport.InsertListRow(t, db, "Want to Buy")
	listID2 := testsupport.InsertListRow(t, db, "Nightstand")
	bookID1 := testsupport.InsertBookRow(t, db, "Dune", nil)
	bookID2 := testsupport.InsertBookRow(t, db, "Foundation", nil)
	testsupport.InsertBookListRow(t, db, listID1, bookID1)
	testsupport.InsertBookListRow(t, db, listID2, bookID2)

	bookList, err := repository.ListByListID(ctx, listID1)
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
