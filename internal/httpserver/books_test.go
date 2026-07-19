package httpserver_test

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/testsupport"
	"github.com/google/uuid"
)

// ── Create ────────────────────────────────────────────────────────────────────

func TestBookAPICreate(t *testing.T) {
	app := newTestApp(t)

	body := bytes.NewBufferString(`{
		"title":"Dune",
		"isbn":"9780441172719",
		"language":"en",
		"publisher":"Chilton",
		"edition":"1st",
		"format":"paperback",
		"purchased_at":"2025-06-15",
		"pages":412,
		"notes":"Classic",
		"published_year":1965,
		"published_month":8,
		"published_day":1}`)

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

	isbn := "9780441172719"
	language := "en"
	publisher := "Chilton"
	edition := "1st"
	format := "paperback"
	purchasedAt := "2025-06-15"
	pages := 412
	notes := "Classic"
	publishedYear := 1965
	publishedMonth := 8
	publishedDay := 1

	testsupport.AssertBookCount(t, app.db, 1)
	testsupport.AssertBookRowFields(t, app.db, created.ID, testsupport.BookRowAssertion{
		Title:          "Dune",
		ISBN:           &isbn,
		Language:       &language,
		Publisher:      &publisher,
		Edition:        &edition,
		Format:         &format,
		PurchasedAt:    &purchasedAt,
		Pages:          &pages,
		Notes:          &notes,
		PublishedYear:  &publishedYear,
		PublishedMonth: &publishedMonth,
		PublishedDay:   &publishedDay,
	})
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

func TestBookAPICreateReturnsConflictForDuplicateTitle(t *testing.T) {
	app := newTestApp(t)
	testsupport.InsertBookRow(t, app.db, "Dune", nil)

	req := httptest.NewRequest(http.MethodPost, "/api/books", bytes.NewBufferString(`{"title":"dUnE"}`))
	resp := httptest.NewRecorder()
	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusConflict {
		t.Fatalf("expected status %d, got %d: %s", http.StatusConflict, resp.Code, resp.Body.String())
	}
}

func TestBookAPICreateRejectsInvalidDatesAndNumbers(t *testing.T) {
	for _, body := range []string{
		`{"title":"Bad Purchase","purchased_at":"2025-02-29"}`,
		`{"title":"Bad Pages","pages":0}`,
		`{"title":"Bad Month","published_month":1}`,
		`{"title":"Bad Day","published_year":2023,"published_month":2,"published_day":29}`,
	} {
		t.Run(body, func(t *testing.T) {
			app := newTestApp(t)
			req := httptest.NewRequest(http.MethodPost, "/api/books", bytes.NewBufferString(body))
			resp := httptest.NewRecorder()
			app.handler.ServeHTTP(resp, req)
			if resp.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, resp.Code, resp.Body.String())
			}
		})
	}
}

// ── List ──────────────────────────────────────────────────────────────────────

func TestBookAPIList(t *testing.T) {
	app := newTestApp(t)
	now := "2026-01-02T03:04:05Z"

	id := uuid.NewString()
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
