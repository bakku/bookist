package client_test

import (
	"context"
	"testing"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/optional"
	"bakku.dev/bookist/internal/testsupport"
)

// ── Authors List ──────────────────────────────────────────────────────────────

func TestAuthorsList(t *testing.T) {
	c, db := newTestClient(t)
	herbertID := testsupport.InsertAuthorRow(t, db, "Frank Herbert")
	testsupport.InsertAuthorRow(t, db, "Octavia Butler")

	listed, err := c.Authors.List(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	if len(listed) != 2 {
		t.Fatalf("expected 2 authors, got %#v", listed)
	}

	found, err := c.Authors.List(context.Background(), "herbert")
	if err != nil {
		t.Fatal(err)
	}
	if len(found) != 1 || found[0].ID != herbertID || found[0].Name != "Frank Herbert" {
		t.Fatalf("expected only Frank Herbert, got %#v", found)
	}
}

// ── Authors Create ────────────────────────────────────────────────────────────

func TestAuthorsCreatePersistsAuthor(t *testing.T) {
	c, db := newTestClient(t)

	created, err := c.Authors.Create(context.Background(), authors.CreateAuthorRequest{Name: "Frank Herbert"})
	if err != nil {
		t.Fatal(err)
	}

	if created.ID == 0 || created.Name != "Frank Herbert" {
		t.Fatalf("unexpected created author %#v", created)
	}
	testsupport.AssertAuthorRow(t, db, created.ID, "Frank Herbert")
}

func TestAuthorsCreateRejectsBlankName(t *testing.T) {
	c, db := newTestClient(t)

	_, err := c.Authors.Create(context.Background(), authors.CreateAuthorRequest{Name: " "})

	assertClientError(t, err, 400, "name_required", "name")
	testsupport.AssertSQLCount(t, db, 0, `SELECT COUNT(*) FROM authors`)
}

// ── Authors Update ────────────────────────────────────────────────────────────

func TestAuthorsUpdatePersistsName(t *testing.T) {
	c, db := newTestClient(t)
	id := testsupport.InsertAuthorRow(t, db, "Ursula Le Guin")

	updated, err := c.Authors.Update(context.Background(), id, authors.UpdateAuthorRequest{Name: optional.Set("Ursula K. Le Guin")})
	if err != nil {
		t.Fatal(err)
	}

	if updated.ID != id || updated.Name != "Ursula K. Le Guin" {
		t.Fatalf("unexpected updated author %#v", updated)
	}
	testsupport.AssertAuthorRow(t, db, id, "Ursula K. Le Guin")
}

// ── Authors Delete ────────────────────────────────────────────────────────────

func TestAuthorsDeleteRemovesAuthor(t *testing.T) {
	c, db := newTestClient(t)
	id := testsupport.InsertAuthorRow(t, db, "Frank Herbert")

	if err := c.Authors.Delete(context.Background(), id); err != nil {
		t.Fatal(err)
	}

	testsupport.AssertSQLCount(t, db, 0, `SELECT COUNT(*) FROM authors`)
}

func TestAuthorsDeleteMissingAuthorReturnsNotFound(t *testing.T) {
	c, _ := newTestClient(t)

	err := c.Authors.Delete(context.Background(), 999)

	assertClientError(t, err, 404, "author_not_found", "")
}
