package api_test

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"bakku.dev/bookist/internal/reads"
	"bakku.dev/bookist/internal/testsupport"
)

// ── Create ────────────────────────────────────────────────────────────────────

func TestReadAPICreate(t *testing.T) {
	app := newTestApp(t)

	bookID := testsupport.InsertBookRow(t, app.db, "Dune", nil)
	body := bytes.NewBufferString(`{
		"abandoned_at":"2026-01-03",
		"rating":4.5,
		"notes":"Excellent"
	}`)

	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/books/%d/reads", bookID), body)
	req.Header.Set("Content-Type", "application/json")
	resp := httptest.NewRecorder()

	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusCreated {
		t.Fatalf("expected status %d, got %d: %s", http.StatusCreated, resp.Code, resp.Body.String())
	}

	var response map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&response); err != nil {
		t.Fatal(err)
	}

	if response["book_id"] != float64(bookID) || response["abandoned_at"] != "2026-01-03" || response["rating"] != 4.5 || response["notes"] != "Excellent" {
		t.Fatalf("unexpected response: %#v", response)
	}

	if response["created_at"] == nil {
		t.Fatal("created_at must be exposed")
	}

	if response["updated_at"] == nil {
		t.Fatal("updated_at must be exposed")
	}

	testsupport.AssertReadRow(t, app.db, int64(response["id"].(float64)), testsupport.ReadRowAssertion{
		BookID:      bookID,
		AbandonedAt: new("2026-01-03"),
		Rating:      new(4.5),
		Notes:       new("Excellent"),
	})
}

func TestReadAPICreateReturns404ForUnknownBook(t *testing.T) {
	app := newTestApp(t)
	req := httptest.NewRequest(http.MethodPost, "/api/books/999999/reads", bytes.NewBufferString(`{}`))
	resp := httptest.NewRecorder()

	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, resp.Code, resp.Body.String())
	}
}

func TestReadAPIRejectsInvalidBookID(t *testing.T) {
	for _, path := range []string{"/api/books/not-a-number/reads", "/api/books/0/reads", "/api/books/-1/reads"} {
		t.Run(path, func(t *testing.T) {
			app := newTestApp(t)
			req := httptest.NewRequest(http.MethodGet, path, nil)
			resp := httptest.NewRecorder()
			app.handler.ServeHTTP(resp, req)
			if resp.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, resp.Code, resp.Body.String())
			}
		})
	}
}

func TestReadAPICreateRejectsInvalidValues(t *testing.T) {
	tests := []string{
		`{"started_at":"not-a-date"}`,
		`{"started_at":"2026-02-02","finished_at":"2026-02-01"}`,
		`{"abandoned_at":"2026-02-30"}`,
		`{"started_at":"2026-02-02","abandoned_at":"2026-02-01"}`,
		`{"finished_at":"2026-02-02","abandoned_at":"2026-02-03"}`,
		`{"rating":0.5}`,
		`{"rating":4.2}`,
		`{"rating":5.5}`,
	}
	for _, body := range tests {
		t.Run(body, func(t *testing.T) {
			app := newTestApp(t)
			bookID := testsupport.InsertBookRow(t, app.db, "Dune", nil)
			req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/books/%d/reads", bookID), bytes.NewBufferString(body))
			resp := httptest.NewRecorder()

			app.handler.ServeHTTP(resp, req)
			if resp.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, resp.Code, resp.Body.String())
			}
			var count int
			if err := app.db.QueryRow(`SELECT COUNT(*) FROM reads`).Scan(&count); err != nil {
				t.Fatal(err)
			}
			if count != 0 {
				t.Fatalf("expected no persisted reads, got %d", count)
			}
		})
	}
}

// ── List ──────────────────────────────────────────────────────────────────────

func TestReadAPIList(t *testing.T) {
	app := newTestApp(t)

	bookID := testsupport.InsertBookRow(t, app.db, "Dune", nil)
	olderID := int64(100)
	newerID := int64(101)

	testsupport.InsertReadRow(t, app.db, testsupport.ReadRow{
		ID: olderID, BookID: bookID, StartedAt: new("2025-01-01"), FinishedAt: new("2025-01-03"),
		Rating: new(4.0), Notes: new("Good"), CreatedAt: "2026-01-01T00:00:00Z",
	})

	testsupport.InsertReadRow(t, app.db, testsupport.ReadRow{
		ID: newerID, BookID: bookID, StartedAt: new("2026-01-01"), AbandonedAt: new("2026-01-03"),
		Rating: new(4.5), Notes: new("Excellent"), CreatedAt: "2026-01-02T00:00:00Z",
	})

	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/books/%d/reads", bookID), nil)
	resp := httptest.NewRecorder()

	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var listed []reads.Read

	if err := json.NewDecoder(resp.Body).Decode(&listed); err != nil {
		t.Fatal(err)
	}

	if len(listed) != 2 {
		t.Fatalf("expected 2 reads, got %d", len(listed))
	}

	newer := listed[0]

	if newer.ID != newerID || newer.BookID != bookID {
		t.Fatalf("expected newer read IDs %d and %d, got %#v", newerID, bookID, newer)
	}

	if newer.StartedAt == nil || *newer.StartedAt != "2026-01-01" || newer.FinishedAt != nil ||
		newer.AbandonedAt == nil || *newer.AbandonedAt != "2026-01-03" {
		t.Fatalf("unexpected newer read dates: %#v", newer)
	}

	if newer.Rating == nil || *newer.Rating != 4.5 || newer.Notes == nil || *newer.Notes != "Excellent" {
		t.Fatalf("unexpected newer read metadata: %#v", newer)
	}

	newerTimestamp := time.Date(2026, 1, 2, 0, 0, 0, 0, time.UTC)

	if !newer.CreatedAt.Equal(newerTimestamp) || !newer.UpdatedAt.Equal(newerTimestamp) {
		t.Fatalf("unexpected newer read timestamps: %#v", newer)
	}

	older := listed[1]

	if older.ID != olderID || older.BookID != bookID {
		t.Fatalf("expected older read IDs %d and %d, got %#v", olderID, bookID, older)
	}

	if older.StartedAt == nil || *older.StartedAt != "2025-01-01" ||
		older.FinishedAt == nil || *older.FinishedAt != "2025-01-03" || older.AbandonedAt != nil {
		t.Fatalf("unexpected older read dates: %#v", older)
	}

	if older.Rating == nil || *older.Rating != 4.0 || older.Notes == nil || *older.Notes != "Good" {
		t.Fatalf("unexpected older read metadata: %#v", older)
	}

	olderTimestamp := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)

	if !older.CreatedAt.Equal(olderTimestamp) || !older.UpdatedAt.Equal(olderTimestamp) {
		t.Fatalf("unexpected older read timestamps: %#v", older)
	}
}

func TestReadAPIListReturns404ForUnknownBook(t *testing.T) {
	app := newTestApp(t)
	req := httptest.NewRequest(http.MethodGet, "/api/books/999999/reads", nil)
	resp := httptest.NewRecorder()

	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, resp.Code)
	}
}

// ── Delete ────────────────────────────────────────────────────────────────────

func TestReadAPIDeleteLeavesBookAndOtherReads(t *testing.T) {
	app := newTestApp(t)

	bookID := testsupport.InsertBookRow(t, app.db, "Dune", nil)

	testsupport.InsertReadRow(t, app.db, testsupport.ReadRow{ID: 100, BookID: bookID, CreatedAt: "2026-01-01T00:00:00Z"})
	testsupport.InsertReadRow(t, app.db, testsupport.ReadRow{ID: 101, BookID: bookID, CreatedAt: "2026-01-02T00:00:00Z"})

	resp := httptest.NewRecorder()
	app.handler.ServeHTTP(resp, httptest.NewRequest(http.MethodDelete, "/api/reads/100", nil))

	if resp.Code != http.StatusNoContent {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNoContent, resp.Code, resp.Body.String())
	}

	testsupport.AssertSQLCount(t, app.db, 0, `SELECT COUNT(*) FROM reads WHERE id = 100`)
	testsupport.AssertSQLCount(t, app.db, 1, `SELECT COUNT(*) FROM reads WHERE id = 101`)
	testsupport.AssertSQLCount(t, app.db, 1, `SELECT COUNT(*) FROM books WHERE id = ?`, bookID)
}

func TestReadAPIDeleteRejectsInvalidID(t *testing.T) {
	for _, id := range []string{"not-a-number", "0", "-1"} {
		t.Run(id, func(t *testing.T) {
			app := newTestApp(t)

			resp := httptest.NewRecorder()

			app.handler.ServeHTTP(resp, httptest.NewRequest(http.MethodDelete, "/api/reads/"+id, nil))

			if resp.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, resp.Code, resp.Body.String())
			}
		})
	}
}

func TestReadAPIDeleteReturns404ForUnknownRead(t *testing.T) {
	app := newTestApp(t)

	resp := httptest.NewRecorder()

	app.handler.ServeHTTP(resp, httptest.NewRequest(http.MethodDelete, "/api/reads/999999", nil))

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, resp.Code, resp.Body.String())
	}
}
