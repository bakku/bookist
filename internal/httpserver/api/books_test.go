package api_test

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/testsupport"
)

// ── Create ────────────────────────────────────────────────────────────────────

func TestBookAPICreate(t *testing.T) {
	app := newTestApp(t)
	authorID := testsupport.InsertAuthorRow(t, app.db, "Frank Herbert")
	image := []byte("\x89PNG\r\n\x1a\ncover")

	body := bytes.NewBufferString(fmt.Sprintf(`{
		"title":"Dune",
		"isbn":"9780441172719",
		"author_ids":[%d],
		"cover":%q,
		"language":"en",
		"publisher":"Chilton",
		"edition":"1st",
		"format":"paperback",
		"purchased_at":"2025-06-15",
		"purchase_price":"12.34 EUR",
		"pages":412,
		"notes":"Classic",
		"summary":"A desert epic",
		"series_name":"Dune",
		"series_position":1.5,
		"location":"Living room",
		"condition":"very_good",
		"acquisition_source":"Bookshop",
		"published_year":1965,
		"published_month":8,
		"published_day":1}`,
		authorID, base64.StdEncoding.EncodeToString(image)))

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

	if created.ID <= 0 {
		t.Fatal("expected created book to have an ID")
	}

	if created.Title != "Dune" {
		t.Fatalf("expected Dune, got %q", created.Title)
	}

	if created.ISBN == nil || *created.ISBN != "9780441172719" {
		t.Fatalf("expected ISBN, got %#v", created.ISBN)
	}

	if created.PurchasePrice == nil || *created.PurchasePrice != "12.34 EUR" {
		t.Fatalf("expected purchase_price, got %#v", created.PurchasePrice)
	}

	if len(created.Authors) != 1 || created.Authors[0].Name != "Frank Herbert" {
		t.Fatalf("expected Frank Herbert, got %#v", created.Authors)
	}

	if created.CoverURL == nil {
		t.Fatal("expected cover_url")
	}

	var coverImageKey string
	if err := app.db.QueryRow(`SELECT cover_image_key FROM books WHERE id = ?`, created.ID).Scan(&coverImageKey); err != nil {
		t.Fatal(err)
	}

	if got, want := *created.CoverURL, "/book-covers/"+coverImageKey; got != want {
		t.Fatalf("expected cover URL %q, got %q", want, got)
	}

	stored, err := os.ReadFile(filepath.Join(app.coverDir, coverImageKey))
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(stored, image) {
		t.Fatalf("expected stored image %v, got %v", image, stored)
	}

	isbn := "9780441172719"
	language := "en"
	publisher := "Chilton"
	edition := "1st"
	format := "paperback"
	purchasedAt := "2025-06-15"
	purchasePrice := "12.34 EUR"
	pages := 412
	notes := "Classic"
	summary := "A desert epic"
	seriesName := "Dune"
	seriesPosition := 1.5
	location := "Living room"
	condition := "very_good"
	acquisitionSource := "Bookshop"
	publishedYear := 1965
	publishedMonth := 8
	publishedDay := 1

	testsupport.AssertSQLCount(t, app.db, 1, `SELECT COUNT(*) FROM books`)
	testsupport.AssertBookRowFields(t, app.db, created.ID, testsupport.BookRowAssertion{
		Title:             "Dune",
		ISBN:              &isbn,
		Language:          &language,
		Publisher:         &publisher,
		Edition:           &edition,
		Format:            &format,
		PurchasedAt:       &purchasedAt,
		PurchasePrice:     &purchasePrice,
		Pages:             &pages,
		Notes:             &notes,
		Summary:           &summary,
		SeriesName:        &seriesName,
		SeriesPosition:    &seriesPosition,
		Location:          &location,
		Condition:         &condition,
		AcquisitionSource: &acquisitionSource,
		PublishedYear:     &publishedYear,
		PublishedMonth:    &publishedMonth,
		PublishedDay:      &publishedDay,
		CoverImageKey:     &coverImageKey,
	})
	testsupport.AssertBookAuthors(t, app.db, created.ID, authorID)
}

func TestBookAPICreateRejectsInvalidCovers(t *testing.T) {
	for _, test := range []struct {
		name   string
		body   func() []byte
		status int
	}{
		{name: "malformed base64", body: func() []byte { return []byte(`{"title":"Dune","cover":"***"}`) }, status: http.StatusBadRequest},
		{name: "unsupported media", body: func() []byte {
			body, _ := json.Marshal(books.CreateBookRequest{Title: "Dune", Cover: new([]byte("plain text"))})
			return body
		}, status: http.StatusUnsupportedMediaType},
		{name: "too large", body: func() []byte {
			image := make([]byte, 10<<20+1)
			copy(image, "\x89PNG\r\n\x1a\n")
			body, _ := json.Marshal(books.CreateBookRequest{Title: "Dune", Cover: &image})
			return body
		}, status: http.StatusRequestEntityTooLarge},
	} {
		t.Run(test.name, func(t *testing.T) {
			app := newTestApp(t)

			req := httptest.NewRequest(http.MethodPost, "/api/books", bytes.NewReader(test.body()))
			resp := httptest.NewRecorder()

			app.handler.ServeHTTP(resp, req)
			if resp.Code != test.status {
				t.Fatalf("expected status %d, got %d: %s", test.status, resp.Code, resp.Body.String())
			}

			testsupport.AssertSQLCount(t, app.db, 0, `SELECT COUNT(*) FROM books`)
		})
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

	testsupport.AssertSQLCount(t, app.db, 0, `SELECT COUNT(*) FROM books`)
}

func TestBookAPICreateRejectsUnknownAuthor(t *testing.T) {
	app := newTestApp(t)

	body := bytes.NewBufferString(`{"title":"Test Book","author_ids":[999999]}`)
	req := httptest.NewRequest(http.MethodPost, "/api/books", body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	app.handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, resp.Code, resp.Body.String())
	}

	testsupport.AssertSQLCount(t, app.db, 0, `SELECT COUNT(*) FROM books`)
}

// ── List ──────────────────────────────────────────────────────────────────────

func TestBookAPIList(t *testing.T) {
	app := newTestApp(t)
	now := "2026-01-02T03:04:05Z"

	id := int64(100)
	coverImageKey := "0123456789abcdef0123456789abcdef.jpg"
	_, err := app.db.ExecContext(context.Background(), `
		INSERT INTO books (id, title, isbn, language, publisher, edition, format,
		                   purchased_at, purchase_price, pages, notes, summary, series_name,
		                   series_position, location, condition, acquisition_source,
		                   published_year, published_month, published_day, cover_image_key,
		                   created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
	`, id, "Dune", "9780441172719", "en", "Chilton", "1st", "paperback", "2025-06-15",
		"12.34 EUR", 412, "Classic", "A desert epic", "Dune", 1.5, "Living room", "very_good", "Bookshop",
		1965, 8, 1, coverImageKey, now, now)
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
		t.Fatalf("expected ID %d, got %d", id, got.ID)
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

	if got.PurchasePrice == nil || *got.PurchasePrice != "12.34 EUR" {
		t.Fatalf("expected purchase_price '12.34 EUR', got %#v", got.PurchasePrice)
	}

	if got.Pages == nil || *got.Pages != 412 {
		t.Fatalf("expected pages 412, got %#v", got.Pages)
	}

	if got.Notes == nil || *got.Notes != "Classic" {
		t.Fatalf("expected notes 'Classic', got %#v", got.Notes)
	}

	if got.Summary == nil || *got.Summary != "A desert epic" {
		t.Fatalf("expected summary, got %#v", got.Summary)
	}

	if got.SeriesName == nil || *got.SeriesName != "Dune" || got.SeriesPosition == nil || *got.SeriesPosition != 1.5 {
		t.Fatalf("expected series metadata, got %#v", got)
	}

	if got.Location == nil || *got.Location != "Living room" {
		t.Fatalf("expected location, got %#v", got.Location)
	}

	if got.Condition == nil || *got.Condition != books.ConditionVeryGood {
		t.Fatalf("expected condition very_good, got %#v", got.Condition)
	}

	if got.AcquisitionSource == nil || *got.AcquisitionSource != "Bookshop" {
		t.Fatalf("expected acquisition source, got %#v", got.AcquisitionSource)
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

	if got.CoverURL == nil || *got.CoverURL != "/book-covers/"+coverImageKey {
		t.Fatalf("expected cover URL, got %#v", got.CoverURL)
	}
}

func TestBookAPISearchesTitlesCaseInsensitively(t *testing.T) {
	app := newTestApp(t)
	testsupport.InsertBookRow(t, app.db, "Dune", nil)
	testsupport.InsertBookRow(t, app.db, "Foundation", nil)

	req := httptest.NewRequest(http.MethodGet, "/api/books?q=+UNE+", nil)
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
		t.Fatalf("expected only Dune, got %#v", listed)
	}
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestBookAPIDeleteCascadesRelationshipsAndReads(t *testing.T) {
	app := newTestApp(t)

	authorID := testsupport.InsertAuthorRow(t, app.db, "Frank Herbert")
	listID := testsupport.InsertListRow(t, app.db, "Favorites")
	bookID := testsupport.InsertBookRow(t, app.db, "Dune", nil)
	otherBookID := testsupport.InsertBookRow(t, app.db, "Foundation", nil)

	testsupport.InsertBookAuthorRow(t, app.db, bookID, authorID)
	testsupport.InsertBookListRow(t, app.db, listID, bookID)
	testsupport.InsertReadRow(t, app.db, testsupport.ReadRow{ID: 100, BookID: bookID, CreatedAt: "2026-01-01T00:00:00Z"})

	req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/api/books/%d", bookID), nil)
	resp := httptest.NewRecorder()
	app.handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusNoContent || resp.Body.Len() != 0 {
		t.Fatalf("expected status %d with empty body, got %d: %s", http.StatusNoContent, resp.Code, resp.Body.String())
	}

	testsupport.AssertSQLCount(t, app.db, 0, `SELECT COUNT(*) FROM books WHERE id = ?`, bookID)
	testsupport.AssertSQLCount(t, app.db, 0, `SELECT COUNT(*) FROM book_authors WHERE book_id = ?`, bookID)
	testsupport.AssertSQLCount(t, app.db, 0, `SELECT COUNT(*) FROM book_lists WHERE book_id = ?`, bookID)
	testsupport.AssertSQLCount(t, app.db, 0, `SELECT COUNT(*) FROM reads WHERE book_id = ?`, bookID)
	testsupport.AssertSQLCount(t, app.db, 1, `SELECT COUNT(*) FROM books WHERE id = ?`, otherBookID)
	testsupport.AssertSQLCount(t, app.db, 1, `SELECT COUNT(*) FROM authors WHERE id = ?`, authorID)
	testsupport.AssertSQLCount(t, app.db, 1, `SELECT COUNT(*) FROM lists WHERE id = ?`, listID)
}

func TestBookAPIDeleteRejectsInvalidID(t *testing.T) {
	for _, id := range []string{"not-a-number", "0", "-1"} {
		t.Run(id, func(t *testing.T) {
			app := newTestApp(t)
			resp := httptest.NewRecorder()
			app.handler.ServeHTTP(resp, httptest.NewRequest(http.MethodDelete, "/api/books/"+id, nil))
			if resp.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, resp.Code, resp.Body.String())
			}
		})
	}
}

func TestBookAPIDeleteReturns404ForUnknownBook(t *testing.T) {
	app := newTestApp(t)
	resp := httptest.NewRecorder()
	app.handler.ServeHTTP(resp, httptest.NewRequest(http.MethodDelete, "/api/books/999999", nil))
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, resp.Code, resp.Body.String())
	}
}
