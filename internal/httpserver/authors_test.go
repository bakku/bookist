package httpserver_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
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
