package httpserver_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/httpserver"
	"bakku.dev/bookist/internal/testsupport"
)

func TestBookAPICreate(t *testing.T) {
	app := newTestApp(t)

	body := bytes.NewBufferString(`{"title":"Dune","isbn":"9780441172719"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/books", body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, resp.Code, resp.Body.String())
	}

	var created books.Book
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.ID == 0 {
		t.Fatal("expected created book to have an ID")
	}
	if created.Title != "Dune" {
		t.Fatalf("expected Dune, got %q", created.Title)
	}
	if created.ISBN == nil || *created.ISBN != "9780441172719" {
		t.Fatalf("expected ISBN, got %#v", created.ISBN)
	}

	testsupport.AssertBookCount(t, app.db, 1)
	testsupport.AssertBookRow(t, app.db, created.ID, "Dune", testsupport.StringPtr("9780441172719"))
}

func TestBookAPIList(t *testing.T) {
	app := newTestApp(t)
	testsupport.InsertBookRow(t, app.db, "Dune", nil)

	req := httptest.NewRequest(http.MethodGet, "/api/books", nil)
	resp := httptest.NewRecorder()
	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}

	var listed []books.Book
	if err := json.NewDecoder(resp.Body).Decode(&listed); err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected 1 book, got %d", len(listed))
	}
	if listed[0].Title != "Dune" {
		t.Fatalf("expected Dune, got %q", listed[0].Title)
	}
}

func TestBookAPICreateRejectsFormBody(t *testing.T) {
	app := newTestApp(t)

	body := bytes.NewBufferString("title=Dune")
	req := httptest.NewRequest(http.MethodPost, "/api/books", body)
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp := httptest.NewRecorder()

	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d", http.StatusBadRequest, resp.Code)
	}
	testsupport.AssertBookCount(t, app.db, 0)
}

func TestIndexListsBooks(t *testing.T) {
	app := newTestApp(t)
	testsupport.InsertBookRow(t, app.db, "Kindred", nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()
	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}
	if !bytes.Contains(resp.Body.Bytes(), []byte("Kindred")) {
		t.Fatalf("expected index response to contain book title, got %s", resp.Body.String())
	}
}

type testApp struct {
	handler http.Handler
	db      *sql.DB
}

func newTestApp(t *testing.T) testApp {
	t.Helper()

	db := testsupport.OpenMigratedDB(t)
	service := books.NewService(books.NewSQLiteRepository(db))
	server, err := httpserver.New(service)
	if err != nil {
		t.Fatal(err)
	}

	return testApp{
		handler: server.Handler(),
		db:      db,
	}
}
