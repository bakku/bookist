package books_test

import (
	"context"
	"errors"
	"fmt"
	"testing"

	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/optional"
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
	purchasePrice := "12.34 EUR"
	pages := 400
	notes := "Great book"
	summary := "A practical Go guide"
	seriesName := "Programming Languages"
	seriesPosition := 1.5
	location := "Office shelf"
	condition := books.ConditionVeryGood
	acquisitionSource := "Local bookstore"
	year := 2024
	month := 6
	day := 15
	coverImageKey := "0123456789abcdef0123456789abcdef.png"

	created, err := repository.Create(ctx, books.CreateBookRequest{
		Title:             "The Go Programming Language",
		ISBN:              &isbn,
		Language:          &lang,
		Publisher:         &pub,
		Edition:           &ed,
		Format:            &format,
		PurchasedAt:       &purchased,
		PurchasePrice:     &purchasePrice,
		Pages:             &pages,
		Notes:             &notes,
		Summary:           &summary,
		SeriesName:        &seriesName,
		SeriesPosition:    &seriesPosition,
		Location:          &location,
		Condition:         &condition,
		AcquisitionSource: &acquisitionSource,
		PublishedYear:     &year,
		PublishedMonth:    &month,
		PublishedDay:      &day,
		CoverImageKey:     &coverImageKey,
	})
	if err != nil {
		t.Fatal(err)
	}

	if created.ID <= 0 {
		t.Fatal("expected created book to have an ID")
	}
	if created.CoverImageKey == nil || *created.CoverImageKey != coverImageKey {
		t.Fatalf("expected cover key %q, got %#v", coverImageKey, created.CoverImageKey)
	}

	f := string(format)
	c := string(condition)
	testsupport.AssertBookRowFields(t, db, created.ID, testsupport.BookRowAssertion{
		Title:             "The Go Programming Language",
		ISBN:              &isbn,
		Language:          &lang,
		Publisher:         &pub,
		Edition:           &ed,
		Format:            &f,
		PurchasedAt:       &purchased,
		PurchasePrice:     &purchasePrice,
		Pages:             &pages,
		Notes:             &notes,
		Summary:           &summary,
		SeriesName:        &seriesName,
		SeriesPosition:    &seriesPosition,
		Location:          &location,
		Condition:         &c,
		AcquisitionSource: &acquisitionSource,
		PublishedYear:     &year,
		PublishedMonth:    &month,
		PublishedDay:      &day,
		CoverImageKey:     &coverImageKey,
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
		AuthorIDs: []int64{authorID},
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

// ── GetByID ───────────────────────────────────────────────────────────────────

func TestSQLiteRepositoryGetByIDReturnsPersistedBook(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	repository := books.NewSQLiteRepository(db)
	coverImageKey := "0123456789abcdef0123456789abcdef.png"
	created, err := repository.Create(context.Background(), books.CreateBookRequest{
		Title: "Dune", CoverImageKey: &coverImageKey,
	})
	if err != nil {
		t.Fatal(err)
	}

	got, err := repository.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != created.ID || got.Title != "Dune" || got.CoverImageKey == nil || *got.CoverImageKey != coverImageKey {
		t.Fatalf("unexpected book: %#v", got)
	}
}

func TestSQLiteRepositoryGetByIDReturnsErrBookNotFound(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	_, err := books.NewSQLiteRepository(db).GetByID(context.Background(), 999999)
	if !errors.Is(err, books.ErrBookNotFound) {
		t.Fatalf("expected ErrBookNotFound, got %v", err)
	}
}

// ── Update ────────────────────────────────────────────────────────────────────

func TestSQLiteRepositoryUpdatePersistsScalarsAndReconcilesAuthors(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	repository := books.NewSQLiteRepository(db)
	author1 := testsupport.InsertAuthorRow(t, db, "Author One")
	author2 := testsupport.InsertAuthorRow(t, db, "Author Two")
	author3 := testsupport.InsertAuthorRow(t, db, "Author Three")
	oldISBN := "old-isbn"
	oldCover := "11111111111111111111111111111111.png"
	created, err := repository.Create(context.Background(), books.CreateBookRequest{
		Title: "Dune", ISBN: &oldISBN, AuthorIDs: []int64{author1, author2}, CoverImageKey: &oldCover,
	})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE books SET created_at = '2000-01-01T00:00:00Z', updated_at = '2000-01-01T00:00:00Z' WHERE id = ?`, created.ID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE book_authors SET created_at = '2000-01-01T00:00:00Z', updated_at = '2000-01-01T00:00:00Z' WHERE book_id = ?`, created.ID); err != nil {
		t.Fatal(err)
	}
	created, err = repository.GetByID(context.Background(), created.ID)
	if err != nil {
		t.Fatal(err)
	}
	var relationshipID int64
	var relationshipCreatedAt, relationshipUpdatedAt string
	if err := db.QueryRow(`SELECT id, created_at, updated_at FROM book_authors WHERE book_id = ? AND author_id = ?`, created.ID, author2).
		Scan(&relationshipID, &relationshipCreatedAt, &relationshipUpdatedAt); err != nil {
		t.Fatal(err)
	}
	newCover := "22222222222222222222222222222222.png"

	result, err := repository.Update(context.Background(), created.ID, books.UpdateBookRequest{
		ISBN:          optionalNull[string](),
		AuthorIDs:     updateValueOf([]int64{author2, author3}),
		CoverImageKey: updateValueOf(newCover),
	})
	if err != nil {
		t.Fatal(err)
	}
	updated := result.Book
	priorCover := result.PriorCoverImageKey
	if updated.Title != "Dune" || updated.ISBN != nil || updated.CoverImageKey == nil || *updated.CoverImageKey != newCover {
		t.Fatalf("unexpected updated book: %#v", updated)
	}
	if priorCover == nil || *priorCover != oldCover {
		t.Fatalf("expected prior cover %q, got %#v", oldCover, priorCover)
	}
	if !updated.CreatedAt.Equal(created.CreatedAt) || !updated.UpdatedAt.After(created.UpdatedAt) {
		t.Fatalf("expected created_at preserved and updated_at advanced: created=%v updated=%v", created, updated)
	}
	testsupport.AssertBookAuthors(t, db, created.ID, author2, author3)
	if len(updated.Authors) != 2 || updated.Authors[0].ID != author3 || updated.Authors[1].ID != author2 {
		t.Fatalf("expected canonical relationship order [%d %d], got %#v", author3, author2, updated.Authors)
	}

	var gotID int64
	var gotCreatedAt, gotUpdatedAt string
	if err := db.QueryRow(`SELECT id, created_at, updated_at FROM book_authors WHERE book_id = ? AND author_id = ?`, created.ID, author2).
		Scan(&gotID, &gotCreatedAt, &gotUpdatedAt); err != nil {
		t.Fatal(err)
	}
	if gotID != relationshipID || gotCreatedAt != relationshipCreatedAt || gotUpdatedAt != relationshipUpdatedAt {
		t.Fatalf("expected unchanged relationship row to be preserved")
	}
}

func TestSQLiteRepositoryUpdateRollsBackBookAndRelationshipsAfterAuthorInsertFails(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	repository := books.NewSQLiteRepository(db)
	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)
	originalAuthorID := testsupport.InsertAuthorRow(t, db, "Original Author")
	replacementAuthorID := testsupport.InsertAuthorRow(t, db, "Replacement Author")
	testsupport.InsertBookAuthorRow(t, db, bookID, originalAuthorID)

	originalUpdatedAt := "2000-01-01T00:00:00Z"
	originalRelationshipCreatedAt := "2000-01-02T00:00:00Z"
	originalRelationshipUpdatedAt := "2000-01-03T00:00:00Z"
	if _, err := db.Exec(`UPDATE books SET updated_at = ? WHERE id = ?`, originalUpdatedAt, bookID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE book_authors SET created_at = ?, updated_at = ? WHERE book_id = ? AND author_id = ?`,
		originalRelationshipCreatedAt, originalRelationshipUpdatedAt, bookID, originalAuthorID); err != nil {
		t.Fatal(err)
	}

	var originalRelationshipID int64
	if err := db.QueryRow(`SELECT id FROM book_authors WHERE book_id = ? AND author_id = ?`, bookID, originalAuthorID).
		Scan(&originalRelationshipID); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(fmt.Sprintf(`
		CREATE TRIGGER reject_replacement_author
		BEFORE INSERT ON book_authors
		WHEN NEW.author_id = %d
		BEGIN
			SELECT RAISE(ABORT, 'replacement rejected');
		END
	`, replacementAuthorID)); err != nil {
		t.Fatal(err)
	}

	_, err := repository.Update(context.Background(), bookID, books.UpdateBookRequest{
		Title:     updateValueOf("Dune Messiah"),
		AuthorIDs: updateValueOf([]int64{replacementAuthorID}),
	})
	if err == nil {
		t.Fatal("expected replacement relationship insert to fail")
	}

	var title, updatedAt string
	if err := db.QueryRow(`SELECT title, updated_at FROM books WHERE id = ?`, bookID).Scan(&title, &updatedAt); err != nil {
		t.Fatal(err)
	}
	if title != "Dune" || updatedAt != originalUpdatedAt {
		t.Fatalf("expected book rollback, got title=%q updated_at=%q", title, updatedAt)
	}

	var relationshipID int64
	var relationshipCreatedAt, relationshipUpdatedAt string
	if err := db.QueryRow(`
		SELECT id, created_at, updated_at
		FROM book_authors
		WHERE book_id = ? AND author_id = ?
	`, bookID, originalAuthorID).Scan(&relationshipID, &relationshipCreatedAt, &relationshipUpdatedAt); err != nil {
		t.Fatal(err)
	}
	if relationshipID != originalRelationshipID || relationshipCreatedAt != originalRelationshipCreatedAt || relationshipUpdatedAt != originalRelationshipUpdatedAt {
		t.Fatalf("expected original relationship rollback, got id=%d created_at=%q updated_at=%q", relationshipID, relationshipCreatedAt, relationshipUpdatedAt)
	}
	testsupport.AssertSQLCount(t, db, 0, `SELECT COUNT(*) FROM book_authors WHERE book_id = ? AND author_id = ?`, bookID, replacementAuthorID)
}

func optionalNull[T any]() optional.Value[T] {
	return optional.Value[T]{Present: true}
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestSQLiteRepositoryDeleteReturnsErrBookNotFoundForUnknownID(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	repository := books.NewSQLiteRepository(db)

	err := repository.Delete(context.Background(), 999999)
	if !errors.Is(err, books.ErrBookNotFound) {
		t.Fatalf("expected ErrBookNotFound, got %v", err)
	}
}

func TestSQLiteRepositoryDeleteCascadesBookRelationships(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	repository := books.NewSQLiteRepository(db)
	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)
	authorID := testsupport.InsertAuthorRow(t, db, "Frank Herbert")
	listID := testsupport.InsertListRow(t, db, "Favorites")
	testsupport.InsertBookAuthorRow(t, db, bookID, authorID)
	testsupport.InsertBookListRow(t, db, listID, bookID)
	testsupport.InsertReadRow(t, db, testsupport.ReadRow{ID: 1, BookID: bookID, CreatedAt: "2026-01-01T00:00:00Z"})

	if err := repository.Delete(context.Background(), bookID); err != nil {
		t.Fatal(err)
	}

	var booksCount, bookAuthorsCount, bookListsCount, readsCount, authorsCount, listsCount int
	err := db.QueryRow(`
		SELECT
			(SELECT count(*) FROM books WHERE id = ?),
			(SELECT count(*) FROM book_authors WHERE book_id = ?),
			(SELECT count(*) FROM book_lists WHERE book_id = ?),
			(SELECT count(*) FROM reads WHERE book_id = ?),
			(SELECT count(*) FROM authors WHERE id = ?),
			(SELECT count(*) FROM lists WHERE id = ?)
	`, bookID, bookID, bookID, bookID, authorID, listID).Scan(
		&booksCount, &bookAuthorsCount, &bookListsCount, &readsCount, &authorsCount, &listsCount,
	)
	if err != nil {
		t.Fatal(err)
	}
	if booksCount != 0 || bookAuthorsCount != 0 || bookListsCount != 0 || readsCount != 0 {
		t.Fatalf("expected book and relationships deleted, got books=%d book_authors=%d book_lists=%d reads=%d", booksCount, bookAuthorsCount, bookListsCount, readsCount)
	}
	if authorsCount != 1 || listsCount != 1 {
		t.Fatalf("expected author and list retained, got authors=%d lists=%d", authorsCount, listsCount)
	}
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
	purchasePrice := "12.34 EUR"
	pages := 400
	notes := "Great book"
	summary := "A practical Go guide"
	seriesName := "Programming Languages"
	seriesPosition := 1.5
	location := "Office shelf"
	condition := books.ConditionVeryGood
	acquisitionSource := "Local bookstore"
	year := 2024
	month := 6
	day := 15

	created, err := repository.Create(ctx, books.CreateBookRequest{
		Title:             "The Go Programming Language",
		ISBN:              &isbn,
		Language:          &lang,
		Publisher:         &pub,
		Edition:           &ed,
		Format:            &format,
		PurchasedAt:       &purchased,
		PurchasePrice:     &purchasePrice,
		Pages:             &pages,
		Notes:             &notes,
		Summary:           &summary,
		SeriesName:        &seriesName,
		SeriesPosition:    &seriesPosition,
		Location:          &location,
		Condition:         &condition,
		AcquisitionSource: &acquisitionSource,
		PublishedYear:     &year,
		PublishedMonth:    &month,
		PublishedDay:      &day,
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
		t.Fatalf("expected ID %d, got %d", created.ID, got.ID)
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

	if got.PurchasePrice == nil || *got.PurchasePrice != purchasePrice {
		t.Fatalf("expected PurchasePrice %q, got %#v", purchasePrice, got.PurchasePrice)
	}

	if got.Pages == nil || *got.Pages != pages {
		t.Fatalf("expected Pages %d, got %#v", pages, got.Pages)
	}

	if got.Notes == nil || *got.Notes != notes {
		t.Fatalf("expected Notes %q, got %#v", notes, got.Notes)
	}

	if got.Summary == nil || *got.Summary != summary {
		t.Fatalf("expected Summary %q, got %#v", summary, got.Summary)
	}

	if got.SeriesName == nil || *got.SeriesName != seriesName {
		t.Fatalf("expected SeriesName %q, got %#v", seriesName, got.SeriesName)
	}

	if got.SeriesPosition == nil || *got.SeriesPosition != seriesPosition {
		t.Fatalf("expected SeriesPosition %v, got %#v", seriesPosition, got.SeriesPosition)
	}

	if got.Location == nil || *got.Location != location {
		t.Fatalf("expected Location %q, got %#v", location, got.Location)
	}

	if got.Condition == nil || *got.Condition != condition {
		t.Fatalf("expected Condition %q, got %#v", condition, got.Condition)
	}

	if got.AcquisitionSource == nil || *got.AcquisitionSource != acquisitionSource {
		t.Fatalf("expected AcquisitionSource %q, got %#v", acquisitionSource, got.AcquisitionSource)
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

// ── Search ────────────────────────────────────────────────────────────────────

func TestSQLiteRepositorySearchMatchesTitlesCaseInsensitively(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := books.NewSQLiteRepository(db)
	id1 := testsupport.InsertBookRow(t, db, "Dune", nil)
	id2 := testsupport.InsertBookRow(t, db, "Dune Messiah", nil)
	id3 := testsupport.InsertBookRow(t, db, "Children of Dune", nil)
	testsupport.InsertBookRow(t, db, "Foundation", nil)
	if _, err := db.Exec(`UPDATE books SET updated_at = '2026-01-03T00:00:00Z' WHERE id = ?`, id3); err != nil {
		t.Fatal(err)
	}

	matched, err := repository.Search(ctx, "dUnE")
	if err != nil {
		t.Fatal(err)
	}
	if len(matched) != 3 {
		t.Fatalf("expected 3 matching books, got %#v", matched)
	}
	if matched[0].ID != id3 || matched[1].ID != id1 || matched[2].ID != id2 {
		t.Fatalf("expected updated-at and ID ordering [%d %d %d], got [%d %d %d]", id3, id1, id2, matched[0].ID, matched[1].ID, matched[2].ID)
	}
}

func TestSQLiteRepositorySearchTreatsSQLWildcardsLiterally(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := books.NewSQLiteRepository(db)
	testsupport.InsertBookRow(t, db, "100% Complete", nil)
	testsupport.InsertBookRow(t, db, "Under_score", nil)
	testsupport.InsertBookRow(t, db, "Dune", nil)

	tests := []struct {
		query string
		want  string
	}{
		{query: "%", want: "100% Complete"},
		{query: "_", want: "Under_score"},
	}

	for _, test := range tests {
		t.Run(test.query, func(t *testing.T) {
			matched, err := repository.Search(ctx, test.query)
			if err != nil {
				t.Fatal(err)
			}
			if len(matched) != 1 || matched[0].Title != test.want {
				t.Fatalf("expected only %q, got %#v", test.want, matched)
			}
		})
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
	if _, err := db.Exec(`UPDATE books SET summary = 'Desert epic', series_position = 1.5 WHERE id = ?`, bookID1); err != nil {
		t.Fatal(err)
	}
	if _, err := db.Exec(`UPDATE book_lists SET updated_at = '2026-01-03T00:00:00Z' WHERE book_id = ?`, bookID1); err != nil {
		t.Fatal(err)
	}

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
	if bookList[0].Summary == nil || *bookList[0].Summary != "Desert epic" ||
		bookList[0].SeriesPosition == nil || *bookList[0].SeriesPosition != 1.5 {
		t.Fatalf("expected extended metadata for Dune, got %#v", bookList[0])
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

func TestSQLiteRepositorySearchByListIDFiltersWithinList(t *testing.T) {
	ctx := context.Background()
	db := testsupport.OpenMigratedDB(t)
	repository := books.NewSQLiteRepository(db)
	listID := testsupport.InsertListRow(t, db, "Nightstand")
	otherListID := testsupport.InsertListRow(t, db, "Archive")
	duneID := testsupport.InsertBookRow(t, db, "Dune", nil)
	duneMessiahID := testsupport.InsertBookRow(t, db, "Dune Messiah", nil)
	childrenID := testsupport.InsertBookRow(t, db, "Children of Dune", nil)
	foundationID := testsupport.InsertBookRow(t, db, "Foundation", nil)
	otherDuneID := testsupport.InsertBookRow(t, db, "The Dune Encyclopedia", nil)
	testsupport.InsertBookListRow(t, db, listID, duneID)
	testsupport.InsertBookListRow(t, db, listID, duneMessiahID)
	testsupport.InsertBookListRow(t, db, listID, childrenID)
	testsupport.InsertBookListRow(t, db, listID, foundationID)
	testsupport.InsertBookListRow(t, db, otherListID, otherDuneID)
	if _, err := db.Exec(`UPDATE book_lists SET updated_at = '2026-01-03T00:00:00Z' WHERE list_id = ? AND book_id = ?`, listID, childrenID); err != nil {
		t.Fatal(err)
	}

	matched, err := repository.SearchByListID(ctx, listID, "DUNE")
	if err != nil {
		t.Fatal(err)
	}
	if len(matched) != 3 {
		t.Fatalf("expected 3 matching books from Nightstand, got %#v", matched)
	}
	if matched[0].ID != childrenID || matched[1].ID != duneID || matched[2].ID != duneMessiahID {
		t.Fatalf("expected relationship ordering [%d %d %d], got [%d %d %d]", childrenID, duneID, duneMessiahID, matched[0].ID, matched[1].ID, matched[2].ID)
	}
}
