package books_test

import (
	"context"
	"errors"
	"testing"

	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/optional"
	"bakku.dev/bookist/internal/testsupport"
)

// ── Author Constraints ────────────────────────────────────────────────────────

func TestCreateMissingAuthor(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	authorID := testsupport.InsertAuthorRow(t, db, "Frank Herbert")
	const missingAuthorID = int64(999999)

	_, err := books.NewSQLiteRepository(db).Create(context.Background(), books.CreateBookRequest{
		Title: "Dune", AuthorIDs: []int64{authorID, missingAuthorID},
	})
	if !errors.Is(err, books.ErrAuthorNotFound) {
		t.Fatalf("expected ErrAuthorNotFound, got %v", err)
	}

	// A failed relationship write must roll back the book and earlier links.
	testsupport.AssertSQLCount(t, db, 0, `SELECT count(*) FROM books`)
	testsupport.AssertSQLCount(t, db, 0, `SELECT count(*) FROM book_authors`)
}

func TestUpdateMissingAuthor(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)
	authorID := testsupport.InsertAuthorRow(t, db, "Frank Herbert")
	testsupport.InsertBookAuthorRow(t, db, bookID, authorID)
	const missingAuthorID = int64(999999)

	_, err := books.NewSQLiteRepository(db).Update(context.Background(), bookID, books.UpdateBookRequest{
		Title:     optional.Value[string]{Present: true, Value: new("Changed")},
		AuthorIDs: optional.Value[[]int64]{Present: true, Value: new([]int64{missingAuthorID})},
	})
	if !errors.Is(err, books.ErrAuthorNotFound) {
		t.Fatalf("expected ErrAuthorNotFound, got %v", err)
	}

	// Failed replacement must preserve the original book and author link.
	testsupport.AssertSQLCount(t, db, 1, `SELECT count(*) FROM books WHERE id = ? AND title = 'Dune'`, bookID)
	testsupport.AssertSQLCount(t, db, 1, `SELECT count(*) FROM book_authors WHERE book_id = ? AND author_id = ?`, bookID, authorID)
}
