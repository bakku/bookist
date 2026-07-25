package httpserver_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

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

	var startedAt, finishedAt, abandonedAt, notes sql.NullString
	var rating sql.NullFloat64
	var createdAt, updatedAt string

	err := app.db.QueryRowContext(context.Background(), `
		SELECT started_at, finished_at, abandoned_at, rating, notes, created_at, updated_at
		FROM reads WHERE id = ?
	`, response["id"]).Scan(&startedAt, &finishedAt, &abandonedAt, &rating, &notes, &createdAt, &updatedAt)

	if err != nil {
		t.Fatal(err)
	}
	if startedAt.Valid || finishedAt.Valid || abandonedAt.String != "2026-01-03" || rating.Float64 != 4.5 || notes.String != "Excellent" {
		t.Fatal("unexpected persisted values")
	}
	if createdAt == "" || updatedAt == "" {
		t.Fatal("expected backend-generated timestamps")
	}
}

func TestReadAPICreateRejectsClientTimestamp(t *testing.T) {
	app := newTestApp(t)
	bookID := testsupport.InsertBookRow(t, app.db, "Dune", nil)
	body := bytes.NewBufferString(`{"created_at":"2026-01-01T00:00:00Z"}`)
	req := httptest.NewRequest(http.MethodPost, fmt.Sprintf("/api/books/%d/reads", bookID), body)
	resp := httptest.NewRecorder()

	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, resp.Code, resp.Body.String())
	}
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

	var raw []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		t.Fatal(err)
	}
	if len(raw) != 2 || raw[0]["id"] != float64(newerID) || raw[1]["id"] != float64(olderID) {
		t.Fatalf("expected newest reads first, got %#v", raw)
	}
	if raw[0]["abandoned_at"] != "2026-01-03" || raw[0]["finished_at"] != nil || raw[1]["abandoned_at"] != nil {
		t.Fatalf("unexpected terminal dates: %#v", raw)
	}
	for _, item := range raw {
		for _, field := range []string{"id", "book_id", "started_at", "finished_at", "abandoned_at", "rating", "notes", "created_at", "updated_at"} {
			if _, exists := item[field]; !exists {
				t.Fatalf("expected field %q in %#v", field, item)
			}
		}
	}
}

func TestReadAPIListReturnsEmptyArray(t *testing.T) {
	app := newTestApp(t)
	bookID := testsupport.InsertBookRow(t, app.db, "Dune", nil)
	req := httptest.NewRequest(http.MethodGet, fmt.Sprintf("/api/books/%d/reads", bookID), nil)
	resp := httptest.NewRecorder()

	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || resp.Body.String() != "[]\n" {
		t.Fatalf("expected empty JSON array, got status %d: %s", resp.Code, resp.Body.String())
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
	assertSQLCount(t, app.db, 0, `SELECT COUNT(*) FROM reads WHERE id = 100`)
	assertSQLCount(t, app.db, 1, `SELECT COUNT(*) FROM reads WHERE id = 101`)
	assertSQLCount(t, app.db, 1, `SELECT COUNT(*) FROM books WHERE id = ?`, bookID)
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
