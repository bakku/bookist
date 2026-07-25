package httpserver_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/testsupport"
)

// ── Create ────────────────────────────────────────────────────────────────────

func TestAuthorAPICreate(t *testing.T) {
	app := newTestApp(t)

	body := bytes.NewBufferString(`{"name":"Jane Austen"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/authors", body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, resp.Code, resp.Body.String())
	}

	var created authors.Author
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.ID <= 0 {
		t.Fatal("expected created author to have an ID")
	}
	if created.Name != "Jane Austen" {
		t.Fatalf("expected Jane Austen, got %q", created.Name)
	}

	testsupport.AssertAuthorCount(t, app.db, 1)
	testsupport.AssertAuthorRow(t, app.db, created.ID, "Jane Austen")
}

// ── Update ────────────────────────────────────────────────────────────────────

func TestAuthorAPIUpdate(t *testing.T) {
	app := newTestApp(t)
	authorID := testsupport.InsertAuthorRow(t, app.db, "Ursula Le Guin")

	resp := patchJSON(t, app.handler, fmt.Sprintf("/api/authors/%d", authorID), `{"name":" Ursula K. Le Guin "}`)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}
	var updated authors.Author
	if err := json.NewDecoder(resp.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	if updated.ID != authorID || updated.Name != "Ursula K. Le Guin" {
		t.Fatalf("unexpected response: %#v", updated)
	}
	testsupport.AssertAuthorRow(t, app.db, authorID, "Ursula K. Le Guin")
}

func TestAuthorAPIUpdateRejectsInvalidRequests(t *testing.T) {
	for _, test := range []struct {
		name   string
		path   string
		body   string
		status int
	}{
		{name: "invalid ID", path: "/api/authors/-1", body: `{"name":"New"}`, status: http.StatusBadRequest},
		{name: "empty", path: "/api/authors/1", body: `{}`, status: http.StatusBadRequest},
		{name: "unknown field", path: "/api/authors/1", body: `{"books":[]}`, status: http.StatusBadRequest},
		{name: "null name", path: "/api/authors/1", body: `{"name":null}`, status: http.StatusBadRequest},
		{name: "blank name", path: "/api/authors/1", body: `{"name":" "}`, status: http.StatusBadRequest},
		{name: "not found", path: "/api/authors/999999", body: `{"name":"New"}`, status: http.StatusNotFound},
	} {
		t.Run(test.name, func(t *testing.T) {
			app := newTestApp(t)
			testsupport.InsertAuthorRow(t, app.db, "Original")
			resp := patchJSON(t, app.handler, test.path, test.body)
			if resp.Code != test.status {
				t.Fatalf("expected status %d, got %d: %s", test.status, resp.Code, resp.Body.String())
			}
		})
	}
}

func TestAuthorAPIUpdateReturns500ForUnexpectedError(t *testing.T) {
	app := newTestApp(t)
	authorID := testsupport.InsertAuthorRow(t, app.db, "Original")
	if _, err := app.db.Exec(`
		CREATE TRIGGER reject_author_update BEFORE UPDATE ON authors
		BEGIN
			SELECT RAISE(ABORT, 'update rejected');
		END
	`); err != nil {
		t.Fatal(err)
	}

	resp := patchJSON(t, app.handler, fmt.Sprintf("/api/authors/%d", authorID), `{"name":"New"}`)
	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d: %s", http.StatusInternalServerError, resp.Code, resp.Body.String())
	}
	testsupport.AssertAuthorRow(t, app.db, authorID, "Original")
}

func TestAuthorAPIUpdateRejectsOversizedBody(t *testing.T) {
	app := newTestApp(t)
	authorID := testsupport.InsertAuthorRow(t, app.db, "Original")
	resp := patchJSON(t, app.handler, fmt.Sprintf("/api/authors/%d", authorID), `{"name":"`+strings.Repeat("a", (1<<20)+1)+`"}`)
	if resp.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("expected status %d, got %d: %s", http.StatusRequestEntityTooLarge, resp.Code, resp.Body.String())
	}
}

// ── List ──────────────────────────────────────────────────────────────────────

func TestAuthorAPIList(t *testing.T) {
	app := newTestApp(t)
	testsupport.InsertAuthorRow(t, app.db, "Jane Austen")

	req := httptest.NewRequest(http.MethodGet, "/api/authors", nil)
	resp := httptest.NewRecorder()
	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}

	var listed []authors.Author
	if err := json.NewDecoder(resp.Body).Decode(&listed); err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected 1 author, got %d", len(listed))
	}
	if listed[0].Name != "Jane Austen" {
		t.Fatalf("expected Jane Austen, got %q", listed[0].Name)
	}
}

func TestAuthorAPISearchesNamesCaseInsensitively(t *testing.T) {
	app := newTestApp(t)
	testsupport.InsertAuthorRow(t, app.db, "Jane Austen")
	testsupport.InsertAuthorRow(t, app.db, "Octavia Butler")

	req := httptest.NewRequest(http.MethodGet, "/api/authors?q=AUST", nil)
	resp := httptest.NewRecorder()
	app.handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}

	var listed []authors.Author
	if err := json.NewDecoder(resp.Body).Decode(&listed); err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].Name != "Jane Austen" {
		t.Fatalf("expected only Jane Austen, got %#v", listed)
	}
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestAuthorAPIDeleteRemovesOnlyAuthorRelationships(t *testing.T) {
	app := newTestApp(t)
	authorID := testsupport.InsertAuthorRow(t, app.db, "Frank Herbert")
	otherAuthorID := testsupport.InsertAuthorRow(t, app.db, "Isaac Asimov")
	bookID := testsupport.InsertBookRow(t, app.db, "Dune", nil)
	testsupport.InsertBookAuthorRow(t, app.db, bookID, authorID)
	testsupport.InsertBookAuthorRow(t, app.db, bookID, otherAuthorID)

	resp := httptest.NewRecorder()
	app.handler.ServeHTTP(resp, httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/authors/%d", authorID), nil))

	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNoContent, resp.Code, resp.Body.String())
	}
	assertSQLCount(t, app.db, 0, `SELECT COUNT(*) FROM authors WHERE id = ?`, authorID)
	assertSQLCount(t, app.db, 0, `SELECT COUNT(*) FROM book_authors WHERE author_id = ?`, authorID)
	assertSQLCount(t, app.db, 1, `SELECT COUNT(*) FROM books WHERE id = ?`, bookID)
	assertSQLCount(t, app.db, 1, `SELECT COUNT(*) FROM book_authors WHERE book_id = ? AND author_id = ?`, bookID, otherAuthorID)
}

func TestAuthorAPIDeleteRejectsInvalidID(t *testing.T) {
	for _, id := range []string{"not-a-number", "0", "-1"} {
		t.Run(id, func(t *testing.T) {
			app := newTestApp(t)
			resp := httptest.NewRecorder()
			app.handler.ServeHTTP(resp, httptest.NewRequest(http.MethodDelete, "/api/authors/"+id, nil))
			if resp.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, resp.Code, resp.Body.String())
			}
		})
	}
}

func TestAuthorAPIDeleteReturns404ForUnknownAuthor(t *testing.T) {
	app := newTestApp(t)
	resp := httptest.NewRecorder()
	app.handler.ServeHTTP(resp, httptest.NewRequest(http.MethodDelete, "/api/authors/999999", nil))
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, resp.Code, resp.Body.String())
	}
}
