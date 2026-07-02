package httpserver_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/httpserver"
	"bakku.dev/bookist/internal/lists"
	"bakku.dev/bookist/internal/testsupport"
	"github.com/google/uuid"
)

type testApp struct {
	handler http.Handler
	db      *sql.DB
}

func newTestApp(t *testing.T) testApp {
	t.Helper()

	db := testsupport.OpenMigratedDB(t)
	authorRepo := authors.NewSQLiteRepository(db)
	authorService := authors.NewService(authorRepo)
	listRepo := lists.NewSQLiteRepository(db)
	listService := lists.NewService(listRepo)
	service := books.NewService(books.NewSQLiteRepository(db), authorRepo)
	server, err := httpserver.New(service, authorService, listService)
	if err != nil {
		t.Fatal(err)
	}

	return testApp{
		handler: server.Handler(),
		db:      db,
	}
}

// ── Book API ──────────────────────────────────────────────────────────────────

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
	if created.ID == "" {
		t.Fatal("expected created book to have an ID")
	}
	if created.Title != "Dune" {
		t.Fatalf("expected Dune, got %q", created.Title)
	}
	if created.ISBN == nil || *created.ISBN != "9780441172719" {
		t.Fatalf("expected ISBN, got %#v", created.ISBN)
	}

	testsupport.AssertBookCount(t, app.db, 1)
	testsupport.AssertBookRow(t, app.db, created.ID, "Dune", new("9780441172719"))
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

func TestBookAPICreateRejectsUnknownAuthor(t *testing.T) {
	app := newTestApp(t)

	body := bytes.NewBufferString(`{"title":"Test Book","author_ids":["` + uuid.NewString() + `"]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/books", body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, resp.Code, resp.Body.String())
	}

	testsupport.AssertBookCount(t, app.db, 0)
}

func TestBookAPICreateWithAuthors(t *testing.T) {
	app := newTestApp(t)
	authorID := testsupport.InsertAuthorRow(t, app.db, "Test Author")

	body := bytes.NewBufferString(`{"title":"Test Book","author_ids":["` + authorID + `"]}`)
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
	if len(created.Authors) != 1 {
		t.Fatalf("expected 1 author, got %d", len(created.Authors))
	}
	if created.Authors[0].Name != "Test Author" {
		t.Fatalf("expected Test Author, got %q", created.Authors[0].Name)
	}

	testsupport.AssertBookAuthors(t, app.db, created.ID, authorID)
}

func TestBookAPICreateWithNewFields(t *testing.T) {
	app := newTestApp(t)

	body := bytes.NewBufferString(`{
		"title":"Full Book",
		"language":"en",
		"publisher":"O'Reilly",
		"edition":"2nd",
		"format":"paperback",
		"purchased_at":"2025-06-15",
		"pages":400,
		"notes":"Great book",
		"published_year":2024,
		"published_month":6,
		"published_day":15}`)

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

	lang := "en"
	pub := "O'Reilly"
	ed := "2nd"
	f := "paperback"
	purchased := "2025-06-15"
	pages := 400
	notes := "Great book"
	year := 2024
	month := 6
	day := 15

	testsupport.AssertBookRowFields(t, app.db, created.ID, testsupport.BookRowAssertion{
		Title:          "Full Book",
		Language:       &lang,
		Publisher:      &pub,
		Edition:        &ed,
		Format:         &f,
		PurchasedAt:    &purchased,
		Pages:          &pages,
		Notes:          &notes,
		PublishedYear:  &year,
		PublishedMonth: &month,
		PublishedDay:   &day,
	})
}

func TestBookAPICreateRejectsInvalidFormat(t *testing.T) {
	app := newTestApp(t)

	body := bytes.NewBufferString(`{"title":"Bad Format","format":"vinyl"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/books", body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, resp.Code, resp.Body.String())
	}

	testsupport.AssertBookCount(t, app.db, 0)
}

func TestBookAPIList(t *testing.T) {
	app := newTestApp(t)
	now := "2026-01-02T03:04:05Z"

	id := "test-list-id"
	_, err := app.db.ExecContext(context.Background(), `
		INSERT INTO books (id, title, isbn, language, publisher, edition, format,
		                   purchased_at, pages, notes, published_year, published_month,
		                   published_day, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, "Dune", "9780441172719", "en", "Chilton", "1st", "paperback", "2025-06-15",
		412, "Classic", 1965, 8, 1, now, now)
	if err != nil {
		t.Fatal(err)
	}

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

	got := listed[0]

	if got.ID != id {
		t.Fatalf("expected ID %s, got %s", id, got.ID)
	}

	if got.Title != "Dune" {
		t.Fatalf("expected Dune, got %q", got.Title)
	}

	if got.ISBN == nil || *got.ISBN != "9780441172719" {
		t.Fatalf("expected ISBN, got %#v", got.ISBN)
	}

	if got.Language == nil || *got.Language != "en" {
		t.Fatalf("expected language 'en', got %#v", got.Language)
	}

	if got.Publisher == nil || *got.Publisher != "Chilton" {
		t.Fatalf("expected publisher 'Chilton', got %#v", got.Publisher)
	}

	if got.Edition == nil || *got.Edition != "1st" {
		t.Fatalf("expected edition '1st', got %#v", got.Edition)
	}

	if got.Format == nil || string(*got.Format) != "paperback" {
		t.Fatalf("expected format 'paperback', got %#v", got.Format)
	}

	if got.PurchasedAt == nil || *got.PurchasedAt != "2025-06-15" {
		t.Fatalf("expected purchased_at '2025-06-15', got %#v", got.PurchasedAt)
	}

	if got.Pages == nil || *got.Pages != 412 {
		t.Fatalf("expected pages 412, got %#v", got.Pages)
	}

	if got.Notes == nil || *got.Notes != "Classic" {
		t.Fatalf("expected notes 'Classic', got %#v", got.Notes)
	}

	if got.PublishedYear == nil || *got.PublishedYear != 1965 {
		t.Fatalf("expected published_year 1965, got %#v", got.PublishedYear)
	}

	if got.PublishedMonth == nil || *got.PublishedMonth != 8 {
		t.Fatalf("expected published_month 8, got %#v", got.PublishedMonth)
	}

	if got.PublishedDay == nil || *got.PublishedDay != 1 {
		t.Fatalf("expected published_day 1, got %#v", got.PublishedDay)
	}
}

// ── Author API ────────────────────────────────────────────────────────────────

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

// ── Index ─────────────────────────────────────────────────────────────────────

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

// ── List API ──────────────────────────────────────────────────────────────────

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
