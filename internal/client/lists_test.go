package client_test

import (
	"context"
	"testing"

	"bakku.dev/bookist/internal/lists"
	"bakku.dev/bookist/internal/optional"
	"bakku.dev/bookist/internal/testsupport"
)

// ── Lists List ────────────────────────────────────────────────────────────────

func TestListsList(t *testing.T) {
	c, db := newTestClient(t)
	nightstandID := testsupport.InsertListRowWithDescription(t, db, "Nightstand", "Reading now")
	testsupport.InsertListRow(t, db, "Want to Buy")

	listed, err := c.Lists.List(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 2 {
		t.Fatalf("expected 2 lists, got %#v", listed)
	}

	found, err := c.Lists.List(context.Background(), "night")
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].ID != nightstandID || found[0].Description == nil || *found[0].Description != "Reading now" {
		t.Fatalf("expected only Nightstand, got %#v", found)
	}
}

// ── Lists Create ──────────────────────────────────────────────────────────────

func TestListsCreatePersistsList(t *testing.T) {
	c, db := newTestClient(t)

	created, err := c.Lists.Create(context.Background(), lists.CreateListRequest{Name: "Nightstand", Description: new("Reading now")})
	if err != nil {
		t.Fatal(err)
	}

	if created.ID == 0 || created.Name != "Nightstand" {
		t.Fatalf("unexpected created list %#v", created)
	}
	testsupport.AssertListRow(t, db, created.ID, "Nightstand", new("Reading now"))
}

// ── Lists Update ──────────────────────────────────────────────────────────────

func TestListsUpdateRenamesAndClearsDescription(t *testing.T) {
	c, db := newTestClient(t)
	id := testsupport.InsertListRowWithDescription(t, db, "Nightstand", "Reading now")

	updated, err := c.Lists.Update(context.Background(), id, lists.UpdateListRequest{Name: optional.Set("Bedside")})
	if err != nil {
		t.Fatal(err)
	}
	if updated.Name != "Bedside" {
		t.Fatalf("unexpected updated list %#v", updated)
	}
	testsupport.AssertListRow(t, db, id, "Bedside", new("Reading now"))

	if _, err := c.Lists.Update(context.Background(), id, lists.UpdateListRequest{Description: optional.Null[string]()}); err != nil {
		t.Fatal(err)
	}
	testsupport.AssertListRow(t, db, id, "Bedside", nil)
}

// ── Lists Delete ──────────────────────────────────────────────────────────────

func TestListsDeleteRemovesList(t *testing.T) {
	c, db := newTestClient(t)
	id := testsupport.InsertListRow(t, db, "Nightstand")

	if err := c.Lists.Delete(context.Background(), id); err != nil {
		t.Fatal(err)
	}

	testsupport.AssertSQLCount(t, db, 0, `SELECT COUNT(*) FROM lists`)
}

// ── Lists Books ───────────────────────────────────────────────────────────────

func TestListsListBooksFiltersByListAndQuery(t *testing.T) {
	c, db := newTestClient(t)
	listID := testsupport.InsertListRow(t, db, "Nightstand")
	duneID := testsupport.InsertBookRow(t, db, "Dune", nil)
	kindredID := testsupport.InsertBookRow(t, db, "Kindred", nil)
	testsupport.InsertBookRow(t, db, "Emma", nil)
	testsupport.InsertBookListRow(t, db, listID, duneID)
	testsupport.InsertBookListRow(t, db, listID, kindredID)

	listed, err := c.Lists.ListBooks(context.Background(), listID, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 2 {
		t.Fatalf("expected the 2 books in the list, got %#v", listed)
	}

	found, err := c.Lists.ListBooks(context.Background(), listID, "kin")
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].ID != kindredID {
		t.Fatalf("expected only Kindred, got %#v", found)
	}
}

func TestListsListBooksMissingListReturnsNotFound(t *testing.T) {
	c, _ := newTestClient(t)

	_, err := c.Lists.ListBooks(context.Background(), 999, "")

	assertClientError(t, err, 404, "list_not_found", "")
}

func TestListsAddBookPersistsRelationship(t *testing.T) {
	c, db := newTestClient(t)
	listID := testsupport.InsertListRow(t, db, "Nightstand")
	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)

	if err := c.Lists.AddBook(context.Background(), listID, bookID); err != nil {
		t.Fatal(err)
	}

	testsupport.AssertBookListRow(t, db, listID, bookID)
}

func TestListsAddBookTwiceReturnsConflict(t *testing.T) {
	c, db := newTestClient(t)
	listID := testsupport.InsertListRow(t, db, "Nightstand")
	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)
	testsupport.InsertBookListRow(t, db, listID, bookID)

	err := c.Lists.AddBook(context.Background(), listID, bookID)

	assertClientError(t, err, 409, "book_already_in_list", "book_id")
}

func TestListsRemoveBookDeletesRelationship(t *testing.T) {
	c, db := newTestClient(t)
	listID := testsupport.InsertListRow(t, db, "Nightstand")
	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)
	testsupport.InsertBookListRow(t, db, listID, bookID)

	if err := c.Lists.RemoveBook(context.Background(), listID, bookID); err != nil {
		t.Fatal(err)
	}

	testsupport.AssertSQLCount(t, db, 0, `SELECT COUNT(*) FROM book_lists`)
	testsupport.AssertSQLCount(t, db, 1, `SELECT COUNT(*) FROM books WHERE id = ?`, bookID)
}

func TestListsRemoveBookNotInListReturnsNotFound(t *testing.T) {
	c, db := newTestClient(t)
	listID := testsupport.InsertListRow(t, db, "Nightstand")
	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)

	err := c.Lists.RemoveBook(context.Background(), listID, bookID)

	assertClientError(t, err, 404, "book_not_in_list", "")
}
