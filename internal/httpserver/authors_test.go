package httpserver_test

import (
	"bytes"
	"encoding/json"
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
	if created.ID == "" {
		t.Fatal("expected created author to have an ID")
	}
	if created.Name != "Jane Austen" {
		t.Fatalf("expected Jane Austen, got %q", created.Name)
	}

	testsupport.AssertAuthorCount(t, app.db, 1)
	testsupport.AssertAuthorRow(t, app.db, created.ID, "Jane Austen")
}

func TestAuthorAPICreateReturnsConflictForDuplicateName(t *testing.T) {
	app := newTestApp(t)
	testsupport.InsertAuthorRow(t, app.db, "Jane Austen")

	req := httptest.NewRequest(http.MethodPost, "/api/authors", bytes.NewBufferString(`{"name":"jane AUSTEN"}`))
	resp := httptest.NewRecorder()
	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d: %s", http.StatusConflict, resp.Code, resp.Body.String())
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
