package client_test

import (
	"context"
	"testing"

	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/optional"
	"bakku.dev/bookist/internal/testsupport"
)

// ── Books List ────────────────────────────────────────────────────────────────

func TestBooksList(t *testing.T) {
	c, db := newTestClient(t)
	duneID := testsupport.InsertBookRow(t, db, "Dune", new("9780441172719"))
	testsupport.InsertBookRow(t, db, "Kindred", nil)

	listed, err := c.Books.List(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 2 {
		t.Fatalf("expected 2 books, got %#v", listed)
	}

	found, err := c.Books.List(context.Background(), "dune")
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].ID != duneID || found[0].Title != "Dune" ||
		found[0].ISBN == nil || *found[0].ISBN != "9780441172719" {
		t.Fatalf("expected only Dune, got %#v", found)
	}
}

func TestBooksListEscapesQuery(t *testing.T) {
	c, db := newTestClient(t)
	id := testsupport.InsertBookRow(t, db, "Lord & Rings", nil)
	testsupport.InsertBookRow(t, db, "Lord", nil)

	listed, err := c.Books.List(context.Background(), "Lord & Rings")
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].ID != id {
		t.Fatalf("expected only %q, got %#v", "Lord & Rings", listed)
	}
}

func TestBooksListReturnsEmptySliceForNoBooks(t *testing.T) {
	c, _ := newTestClient(t)

	listed, err := c.Books.List(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if listed == nil || len(listed) != 0 {
		t.Fatalf("expected empty non-nil slice, got %#v", listed)
	}
}

// ── Books Get ─────────────────────────────────────────────────────────────────

func TestBooksGetReturnsBookWithAuthorsAndCover(t *testing.T) {
	c, db := newTestClient(t)
	testsupport.InsertBookRow(t, db, "Kindred", nil)
	duneID := testsupport.InsertBookRow(t, db, "Dune", new("9780441172719"))
	herbertID := testsupport.InsertAuthorRow(t, db, "Frank Herbert")
	testsupport.InsertBookAuthorRow(t, db, duneID, herbertID)
	coverKey := "0123456789abcdef0123456789abcdef.jpg"
	if _, err := db.Exec(`UPDATE books SET cover_image_key = ? WHERE id = ?`, coverKey, duneID); err != nil {
		t.Fatal(err)
	}

	got, err := c.Books.Get(context.Background(), duneID)
	if err != nil {
		t.Fatal(err)
	}
	if got.ID != duneID || got.Title != "Dune" || got.ISBN == nil || *got.ISBN != "9780441172719" {
		t.Fatalf("unexpected book %#v", got)
	}
	if len(got.Authors) != 1 || got.Authors[0].ID != herbertID {
		t.Fatalf("expected Frank Herbert, got %#v", got.Authors)
	}
	if got.CoverURL == nil || *got.CoverURL != "/book-covers/"+coverKey {
		t.Fatalf("expected cover URL, got %#v", got.CoverURL)
	}
}

func TestBooksGetMissingBookReturnsNotFound(t *testing.T) {
	c, _ := newTestClient(t)

	_, err := c.Books.Get(context.Background(), 999)

	assertClientError(t, err, 404, "book_not_found", "")
}

// ── Books Create ──────────────────────────────────────────────────────────────

func TestBooksCreatePersistsBook(t *testing.T) {
	c, db := newTestClient(t)
	authorID := testsupport.InsertAuthorRow(t, db, "Frank Herbert")
	format := books.FormatPaperback

	created, err := c.Books.Create(context.Background(), books.CreateBookRequest{
		Title:     "Dune",
		ISBN:      new("9780441172719"),
		AuthorIDs: []int64{authorID},
		Format:    &format,
		Pages:     new(412),
	})
	if err != nil {
		t.Fatal(err)
	}

	if created.ID == 0 || created.Title != "Dune" || len(created.Authors) != 1 || created.Authors[0].ID != authorID {
		t.Fatalf("unexpected created book %#v", created)
	}
	testsupport.AssertBookRowFields(t, db, created.ID, testsupport.BookRowAssertion{
		Title:  "Dune",
		ISBN:   new("9780441172719"),
		Format: new("paperback"),
		Pages:  new(412),
	})
	testsupport.AssertBookAuthors(t, db, created.ID, authorID)
}

// ── Books Update ──────────────────────────────────────────────────────────────

func TestBooksUpdateSendsOnlySuppliedFields(t *testing.T) {
	c, db := newTestClient(t)
	id := testsupport.InsertBookRow(t, db, "Dune", new("9780441172719"))

	updated, err := c.Books.Update(context.Background(), id, books.UpdateBookRequest{
		Title: optional.Set("Dune Messiah"),
		Pages: optional.Set(256),
	})
	if err != nil {
		t.Fatal(err)
	}

	if updated.ID != id || updated.Title != "Dune Messiah" {
		t.Fatalf("unexpected updated book %#v", updated)
	}
	testsupport.AssertBookRowFields(t, db, id, testsupport.BookRowAssertion{
		Title: "Dune Messiah",
		ISBN:  new("9780441172719"),
		Pages: new(256),
	})
}

func TestBooksUpdateClearsFieldsWithNull(t *testing.T) {
	c, db := newTestClient(t)
	authorID := testsupport.InsertAuthorRow(t, db, "Frank Herbert")
	id := testsupport.InsertBookRow(t, db, "Dune", new("9780441172719"))
	testsupport.InsertBookAuthorRow(t, db, id, authorID)

	_, err := c.Books.Update(context.Background(), id, books.UpdateBookRequest{
		ISBN:      optional.Null[string](),
		AuthorIDs: optional.Null[[]int64](),
	})
	if err != nil {
		t.Fatal(err)
	}

	testsupport.AssertBookRowFields(t, db, id, testsupport.BookRowAssertion{Title: "Dune"})
	testsupport.AssertBookHasNoAuthors(t, db, id)
}

func TestBooksUpdateWithoutFieldsIsRejected(t *testing.T) {
	c, db := newTestClient(t)
	id := testsupport.InsertBookRow(t, db, "Dune", nil)

	_, err := c.Books.Update(context.Background(), id, books.UpdateBookRequest{})

	assertClientError(t, err, 400, "no_fields_to_update", "")
	testsupport.AssertBookRow(t, db, id, "Dune", nil)
}

// ── Books Delete ──────────────────────────────────────────────────────────────

func TestBooksDeleteRemovesBook(t *testing.T) {
	c, db := newTestClient(t)
	id := testsupport.InsertBookRow(t, db, "Dune", nil)
	testsupport.InsertBookRow(t, db, "Kindred", nil)

	if err := c.Books.Delete(context.Background(), id); err != nil {
		t.Fatal(err)
	}

	testsupport.AssertSQLCount(t, db, 0, `SELECT COUNT(*) FROM books WHERE id = ?`, id)
	testsupport.AssertSQLCount(t, db, 1, `SELECT COUNT(*) FROM books`)
}
