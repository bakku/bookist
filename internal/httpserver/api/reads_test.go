package api_test

import (
	"bytes"
	"context"
	"database/sql"
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

// ── Update ────────────────────────────────────────────────────────────────────

func TestReadAPIUpdateAndClear(t *testing.T) {
	app := newTestApp(t)
	bookID := testsupport.InsertBookRow(t, app.db, "Dune", nil)
	testsupport.InsertReadRow(t, app.db, testsupport.ReadRow{
		ID: 100, BookID: bookID, Rating: new(3.0), Notes: new("Old notes"), CreatedAt: "2026-01-01T00:00:00Z",
	})

	resp := patchJSON(t, app.handler, "/api/reads/100", `{"rating":4.5,"notes":null}`)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}
	var updated map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&updated); err != nil {
		t.Fatal(err)
	}
	if updated["rating"] != 4.5 || updated["notes"] != nil {
		t.Fatalf("unexpected response: %#v", updated)
	}
	var rating sql.NullFloat64
	var notes sql.NullString
	if err := app.db.QueryRow(`SELECT rating, notes FROM reads WHERE id = 100`).Scan(&rating, &notes); err != nil {
		t.Fatal(err)
	}
	if !rating.Valid || rating.Float64 != 4.5 || notes.Valid {
		t.Fatalf("unexpected persisted values: rating=%#v notes=%#v", rating, notes)
	}
}

func TestReadAPIUpdateRejectsInvalidRequests(t *testing.T) {
	for _, test := range []struct {
		name   string
		path   string
		body   string
		status int
	}{
		{name: "invalid ID", path: "/api/reads/zero", body: `{"rating":4}`, status: http.StatusBadRequest},
		{name: "empty", path: "/api/reads/100", body: `{}`, status: http.StatusBadRequest},
		{name: "unknown field", path: "/api/reads/100", body: `{"book_id":2}`, status: http.StatusBadRequest},
		{name: "blank optional", path: "/api/reads/100", body: `{"notes":" "}`, status: http.StatusBadRequest},
		{name: "invalid rating", path: "/api/reads/100", body: `{"rating":4.2}`, status: http.StatusBadRequest},
		{name: "not found", path: "/api/reads/999999", body: `{"rating":4}`, status: http.StatusNotFound},
	} {
		t.Run(test.name, func(t *testing.T) {
			app := newTestApp(t)
			bookID := testsupport.InsertBookRow(t, app.db, "Dune", nil)
			testsupport.InsertReadRow(t, app.db, testsupport.ReadRow{ID: 100, BookID: bookID, CreatedAt: "2026-01-01T00:00:00Z"})
			resp := patchJSON(t, app.handler, test.path, test.body)
			if resp.Code != test.status {
				t.Fatalf("expected status %d, got %d: %s", test.status, resp.Code, resp.Body.String())
			}
		})
	}
}

func TestReadAPIUpdateReturnsValidationErrorAfterConcurrentEdit(t *testing.T) {
	db := testsupport.OpenMigratedDB(t)
	bookID := testsupport.InsertBookRow(t, db, "Dune", nil)
	readID := int64(100)
	testsupport.InsertReadRow(t, db, testsupport.ReadRow{
		ID: readID, BookID: bookID, StartedAt: new("2026-01-01"), CreatedAt: "2026-01-01T00:00:00Z",
	})

	repository := &afterGetReadRepository{
		Repository: reads.NewSQLiteRepository(db),
		afterGet: func() {
			if _, err := db.Exec(`UPDATE reads SET finished_at = '2026-01-02' WHERE id = ?`, readID); err != nil {
				t.Fatal(err)
			}
		},
	}
	app := newTestAppWithReadRepository(t, db, repository)
	resp := patchJSON(t, app.handler, "/api/reads/100", `{"started_at":"2026-01-03"}`)
	if resp.Code != http.StatusBadRequest || resp.Body.String() != "finished_at must not be before started_at\n" {
		t.Fatalf("expected date validation response, got status %d: %s", resp.Code, resp.Body.String())
	}

	var startedAt, finishedAt string
	if err := db.QueryRow(`SELECT started_at, finished_at FROM reads WHERE id = ?`, readID).Scan(&startedAt, &finishedAt); err != nil {
		t.Fatal(err)
	}
	if startedAt != "2026-01-01" || finishedAt != "2026-01-02" {
		t.Fatalf("expected failed patch to preserve the concurrent state, got started_at=%q finished_at=%q", startedAt, finishedAt)
	}
}

type afterGetReadRepository struct {
	reads.Repository
	afterGet func()
}

func (r *afterGetReadRepository) GetByID(ctx context.Context, id int64) (reads.Read, error) {
	result, err := r.Repository.GetByID(ctx, id)
	if err == nil && r.afterGet != nil {
		afterGet := r.afterGet
		r.afterGet = nil
		afterGet()
	}
	return result, err
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
