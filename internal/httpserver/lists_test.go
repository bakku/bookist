package httpserver_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/lists"
	"bakku.dev/bookist/internal/testsupport"
	"github.com/google/uuid"
)

// ── Create ────────────────────────────────────────────────────────────────────

func TestListAPICreate(t *testing.T) {
	app := newTestApp(t)

	body := bytes.NewBufferString(`{"name":"Want to Buy"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/lists", body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, resp.Code, resp.Body.String())
	}

	var created lists.List
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.ID == "" {
		t.Fatal("expected created list to have an ID")
	}
	if created.Name != "Want to Buy" {
		t.Fatalf("expected Want to Buy, got %q", created.Name)
	}

	testsupport.AssertListCount(t, app.db, 1)
	testsupport.AssertListRow(t, app.db, created.ID, "Want to Buy")
}

func TestListAPICreateRejectsBlankName(t *testing.T) {
	app := newTestApp(t)

	body := bytes.NewBufferString(`{"name":"  "}`)
	req := httptest.NewRequest(http.MethodPost, "/api/lists", body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, resp.Code, resp.Body.String())
	}

	testsupport.AssertListCount(t, app.db, 0)
}

func TestListAPICreateWithDescription(t *testing.T) {
	app := newTestApp(t)

	body := bytes.NewBufferString(`{"name":"Nightstand","description":"Currently reading"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/lists", body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, resp.Code, resp.Body.String())
	}

	var created lists.List
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatal(err)
	}
	if created.Description == nil || *created.Description != "Currently reading" {
		t.Fatalf("expected description 'Currently reading', got %#v", created.Description)
	}
}

// ── List ──────────────────────────────────────────────────────────────────────

func TestListAPIList(t *testing.T) {
	app := newTestApp(t)
	testsupport.InsertListRow(t, app.db, "Want to Buy")

	req := httptest.NewRequest(http.MethodGet, "/api/lists", nil)
	resp := httptest.NewRecorder()
	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}

	var listed []lists.List
	if err := json.NewDecoder(resp.Body).Decode(&listed); err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected 1 list, got %d", len(listed))
	}
	if listed[0].Name != "Want to Buy" {
		t.Fatalf("expected Want to Buy, got %q", listed[0].Name)
	}
}

// ── AddBookToList ─────────────────────────────────────────────────────────────

func TestListAPIAddBookToList(t *testing.T) {
	app := newTestApp(t)

	listID := testsupport.InsertListRow(t, app.db, "Want to Buy")
	bookID := testsupport.InsertBookRow(t, app.db, "Dune", nil)

	body := bytes.NewBufferString(`{"book_id":"` + bookID + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/lists/"+listID+"/books", body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNoContent, resp.Code, resp.Body.String())
	}

	testsupport.AssertBookListRow(t, app.db, listID, bookID)
}

func TestListAPIAddBookToListRejectsUnknownBook(t *testing.T) {
	app := newTestApp(t)

	listID := testsupport.InsertListRow(t, app.db, "Want to Buy")
	unknownBookID := uuid.NewString()

	body := bytes.NewBufferString(`{"book_id":"` + unknownBookID + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/lists/"+listID+"/books", body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, resp.Code, resp.Body.String())
	}
}

func TestListAPIAddBookToListReturns404ForUnknownList(t *testing.T) {
	app := newTestApp(t)

	bookID := testsupport.InsertBookRow(t, app.db, "Dune", nil)
	unknownListID := uuid.NewString()

	body := bytes.NewBufferString(`{"book_id":"` + bookID + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/lists/"+unknownListID+"/books", body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, resp.Code, resp.Body.String())
	}
}

func TestListAPIAddBookToListReturns409ForDuplicate(t *testing.T) {
	app := newTestApp(t)

	listID := testsupport.InsertListRow(t, app.db, "Want to Buy")
	bookID := testsupport.InsertBookRow(t, app.db, "Dune", nil)

	body := bytes.NewBufferString(`{"book_id":"` + bookID + `"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/lists/"+listID+"/books", body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected first request to succeed, got %d", resp.Code)
	}

	body = bytes.NewBufferString(`{"book_id":"` + bookID + `"}`)
	req = httptest.NewRequest(http.MethodPost, "/api/lists/"+listID+"/books", body)
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()

	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusConflict {
		t.Fatalf("expected status %d for duplicate, got %d: %s", http.StatusConflict, resp.Code, resp.Body.String())
	}
}

// ── ListByListID ──────────────────────────────────────────────────────────────

func TestListAPIListBooks(t *testing.T) {
	app := newTestApp(t)

	listID := testsupport.InsertListRow(t, app.db, "Want to Buy")
	bookID1 := testsupport.InsertBookRow(t, app.db, "Dune", nil)
	bookID2 := testsupport.InsertBookRow(t, app.db, "Foundation", nil)

	testsupport.InsertBookListRow(t, app.db, listID, bookID1)
	testsupport.InsertBookListRow(t, app.db, listID, bookID2)

	req := httptest.NewRequest(http.MethodGet, "/api/lists/"+listID+"/books", nil)
	resp := httptest.NewRecorder()
	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}

	var bookList []books.Book
	if err := json.NewDecoder(resp.Body).Decode(&bookList); err != nil {
		t.Fatal(err)
	}
	if len(bookList) != 2 {
		t.Fatalf("expected 2 books, got %d", len(bookList))
	}
	if bookList[0].Title != "Dune" {
		t.Fatalf("expected Dune, got %q", bookList[0].Title)
	}
	if bookList[1].Title != "Foundation" {
		t.Fatalf("expected Foundation, got %q", bookList[1].Title)
	}
}

func TestListAPIListBooksReturnsEmptyArrayForEmptyList(t *testing.T) {
	app := newTestApp(t)

	listID := testsupport.InsertListRow(t, app.db, "Want to Buy")

	req := httptest.NewRequest(http.MethodGet, "/api/lists/"+listID+"/books", nil)
	resp := httptest.NewRecorder()
	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}

	var bookList []books.Book
	if err := json.NewDecoder(resp.Body).Decode(&bookList); err != nil {
		t.Fatal(err)
	}
	if bookList == nil {
		t.Fatal("expected non-nil array")
	}
	if len(bookList) != 0 {
		t.Fatalf("expected empty array, got %d books", len(bookList))
	}
}

func TestListAPIListBooksReturns404ForUnknownList(t *testing.T) {
	app := newTestApp(t)

	unknownListID := uuid.NewString()

	req := httptest.NewRequest(http.MethodGet, "/api/lists/"+unknownListID+"/books", nil)
	resp := httptest.NewRecorder()
	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, resp.Code)
	}
}

func TestListAPIListBooksHydratesAuthors(t *testing.T) {
	app := newTestApp(t)

	listID := testsupport.InsertListRow(t, app.db, "Want to Buy")
	bookID := testsupport.InsertBookRow(t, app.db, "Dune", nil)
	authorID := testsupport.InsertAuthorRow(t, app.db, "Frank Herbert")
	testsupport.InsertBookAuthorRow(t, app.db, bookID, authorID)
	testsupport.InsertBookListRow(t, app.db, listID, bookID)

	req := httptest.NewRequest(http.MethodGet, "/api/lists/"+listID+"/books", nil)
	resp := httptest.NewRecorder()
	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}

	var bookList []books.Book
	if err := json.NewDecoder(resp.Body).Decode(&bookList); err != nil {
		t.Fatal(err)
	}
	if len(bookList) != 1 {
		t.Fatalf("expected 1 book, got %d", len(bookList))
	}
	if len(bookList[0].Authors) != 1 {
		t.Fatalf("expected 1 author, got %d", len(bookList[0].Authors))
	}
	if bookList[0].Authors[0].Name != "Frank Herbert" {
		t.Fatalf("expected Frank Herbert, got %q", bookList[0].Authors[0].Name)
	}
}
