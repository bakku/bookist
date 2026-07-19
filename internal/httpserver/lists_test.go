package httpserver_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/lists"
	"bakku.dev/bookist/internal/testsupport"
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
	if created.ID <= 0 {
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

func TestListAPICreateReturnsConflictForDuplicateName(t *testing.T) {
	app := newTestApp(t)
	testsupport.InsertListRow(t, app.db, "Nightstand")

	req := httptest.NewRequest(http.MethodPost, "/api/lists", bytes.NewBufferString(`{"name":"NIGHTSTAND"}`))
	resp := httptest.NewRecorder()
	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d: %s", http.StatusConflict, resp.Code, resp.Body.String())
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

func TestListAPISearchesNamesCaseInsensitively(t *testing.T) {
	app := newTestApp(t)
	testsupport.InsertListRow(t, app.db, "Nightstand")
	testsupport.InsertListRow(t, app.db, "Want to Buy")

	req := httptest.NewRequest(http.MethodGet, "/api/lists?q=NIGHT", nil)
	resp := httptest.NewRecorder()
	app.handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}

	var listed []lists.List
	if err := json.NewDecoder(resp.Body).Decode(&listed); err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].Name != "Nightstand" {
		t.Fatalf("expected only Nightstand, got %#v", listed)
	}
}

// ── AddBookToList ─────────────────────────────────────────────────────────────

func TestListAPIAddBookToList(t *testing.T) {
	app := newTestApp(t)

	listID := testsupport.InsertListRow(t, app.db, "Want to Buy")
	bookID := testsupport.InsertBookRow(t, app.db, "Dune", nil)

	body := bytes.NewBufferString(fmt.Sprintf(`{"book_id":%d}`, bookID))
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/lists/%d/books", listID), body)
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
	unknownBookID := int64(999999)

	body := bytes.NewBufferString(fmt.Sprintf(`{"book_id":%d}`, unknownBookID))
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/lists/%d/books", listID), body)
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
	unknownListID := int64(999999)

	body := bytes.NewBufferString(fmt.Sprintf(`{"book_id":%d}`, bookID))
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/lists/%d/books", unknownListID), body)
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

	body := bytes.NewBufferString(fmt.Sprintf(`{"book_id":%d}`, bookID))
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/lists/%d/books", listID), body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected first request to succeed, got %d", resp.Code)
	}

	body = bytes.NewBufferString(fmt.Sprintf(`{"book_id":%d}`, bookID))
	req = httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/lists/%d/books", listID), body)
	req.Header.Set("Content-Type", "application/json")
	resp = httptest.NewRecorder()

	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusConflict {
		t.Fatalf("expected status %d for duplicate, got %d: %s", http.StatusConflict, resp.Code, resp.Body.String())
	}
}

func TestListAPIAddBookToListRejectsInvalidIDs(t *testing.T) {
	for _, path := range []string{"/api/lists/not-a-number/books", "/api/lists/0/books", "/api/lists/-1/books"} {
		t.Run(path, func(t *testing.T) {
			app := newTestApp(t)
			req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(`{"book_id":1}`))
			resp := httptest.NewRecorder()
			app.handler.ServeHTTP(resp, req)
			if resp.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, resp.Code, resp.Body.String())
			}
		})
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
	if _, err := app.db.Exec(`UPDATE book_lists SET updated_at = '2026-01-03T00:00:00Z' WHERE book_id = ?`, bookID1); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/lists/%d/books", listID), nil)
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

func TestListAPIListBooksSearchesWithinList(t *testing.T) {
	app := newTestApp(t)

	listID := testsupport.InsertListRow(t, app.db, "Nightstand")
	otherListID := testsupport.InsertListRow(t, app.db, "Archive")
	duneID := testsupport.InsertBookRow(t, app.db, "Dune", nil)
	foundationID := testsupport.InsertBookRow(t, app.db, "Foundation", nil)
	otherDuneID := testsupport.InsertBookRow(t, app.db, "Dune Messiah", nil)
	testsupport.InsertBookListRow(t, app.db, listID, duneID)
	testsupport.InsertBookListRow(t, app.db, listID, foundationID)
	testsupport.InsertBookListRow(t, app.db, otherListID, otherDuneID)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/lists/%d/books?q=UNE", listID), nil)
	resp := httptest.NewRecorder()
	app.handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}

	var listed []books.Book
	if err := json.NewDecoder(resp.Body).Decode(&listed); err != nil {
		t.Fatal(err)
	}
	if len(listed) != 1 || listed[0].Title != "Dune" {
		t.Fatalf("expected only Dune from Nightstand, got %#v", listed)
	}
}

func TestListAPIListBooksReturnsEmptyArrayForEmptyList(t *testing.T) {
	app := newTestApp(t)

	listID := testsupport.InsertListRow(t, app.db, "Want to Buy")

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/lists/%d/books", listID), nil)
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

	unknownListID := int64(999999)

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/lists/%d/books", unknownListID), nil)
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

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/lists/%d/books", listID), nil)
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
