package api_test

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

	testsupport.AssertSQLCount(t, app.db, 1, `SELECT COUNT(*) FROM authors`)
	testsupport.AssertAuthorRow(t, app.db, created.ID, "Jane Austen")
}

func TestAuthorAPICreateRejectsBlankName(t *testing.T) {
	app := newTestApp(t)

	resp := postBody(t, app.handler, "/api/authors", "application/json", `{"name":" "}`)
	assertAPIError(t, resp, http.StatusBadRequest, "name_required", "name")
	testsupport.AssertSQLCount(t, app.db, 0, `SELECT COUNT(*) FROM authors`)
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
		code   string
		field  string
	}{
		{name: "invalid ID", path: "/api/authors/-1", body: `{"name":"New"}`, status: http.StatusBadRequest, code: "invalid_id", field: "id"},
		{name: "empty", path: "/api/authors/1", body: `{}`, status: http.StatusBadRequest, code: "no_fields_to_update"},
		{name: "unknown field", path: "/api/authors/1", body: `{"books":[]}`, status: http.StatusBadRequest, code: "invalid_json"},
		{name: "null name", path: "/api/authors/1", body: `{"name":null}`, status: http.StatusBadRequest, code: "name_required", field: "name"},
		{name: "blank name", path: "/api/authors/1", body: `{"name":" "}`, status: http.StatusBadRequest, code: "name_required", field: "name"},
		{name: "not found", path: "/api/authors/999999", body: `{"name":"New"}`, status: http.StatusNotFound, code: "author_not_found"},
	} {
		t.Run(test.name, func(t *testing.T) {
			app := newTestApp(t)
			testsupport.InsertAuthorRow(t, app.db, "Original")
			resp := patchJSON(t, app.handler, test.path, test.body)
			assertAPIError(t, resp, test.status, test.code, test.field)
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
	assertAPIError(t, resp, http.StatusInternalServerError, "internal_error", "")
	if strings.Contains(resp.Body.String(), "update rejected") {
		t.Fatalf("expected internal error details to stay out of the response, got %s", resp.Body.String())
	}
	testsupport.AssertAuthorRow(t, app.db, authorID, "Original")
}

func TestAuthorAPIUpdateRejectsOversizedBody(t *testing.T) {
	app := newTestApp(t)
	authorID := testsupport.InsertAuthorRow(t, app.db, "Original")
	resp := patchJSON(t, app.handler, fmt.Sprintf("/api/authors/%d", authorID), `{"name":"`+strings.Repeat("a", (1<<20)+1)+`"}`)
	assertAPIError(t, resp, http.StatusRequestEntityTooLarge, "body_too_large", "")
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

func TestAuthorAPIListReturnsEmptyArray(t *testing.T) {
	app := newTestApp(t)
	assertEmptyJSONArray(t, app.handler, "/api/authors")
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
	testsupport.AssertSQLCount(t, app.db, 0, `SELECT COUNT(*) FROM authors WHERE id = ?`, authorID)
	testsupport.AssertSQLCount(t, app.db, 0, `SELECT COUNT(*) FROM book_authors WHERE author_id = ?`, authorID)
	testsupport.AssertSQLCount(t, app.db, 1, `SELECT COUNT(*) FROM books WHERE id = ?`, bookID)
	testsupport.AssertSQLCount(t, app.db, 1, `SELECT COUNT(*) FROM book_authors WHERE book_id = ? AND author_id = ?`, bookID, otherAuthorID)
}

func TestAuthorAPIDeleteRejectsInvalidID(t *testing.T) {
	for _, id := range []string{"not-a-number", "0", "-1"} {
		t.Run(id, func(t *testing.T) {
			app := newTestApp(t)
			resp := httptest.NewRecorder()
			app.handler.ServeHTTP(resp, httptest.NewRequest(http.MethodDelete, "/api/authors/"+id, nil))
			assertAPIError(t, resp, http.StatusBadRequest, "invalid_id", "id")
		})
	}
}

func TestAuthorAPIDeleteReturns404ForUnknownAuthor(t *testing.T) {
	app := newTestApp(t)
	resp := httptest.NewRecorder()
	app.handler.ServeHTTP(resp, httptest.NewRequest(http.MethodDelete, "/api/authors/999999", nil))
	assertAPIError(t, resp, http.StatusNotFound, "author_not_found", "")
}
