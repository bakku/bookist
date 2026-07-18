package httpserver_test

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"bakku.dev/bookist/internal/testsupport"
	"github.com/google/uuid"
)

// ── Create ────────────────────────────────────────────────────────────────────

func TestReadAPICreate(t *testing.T) {
	app := newTestApp(t)

	bookID := testsupport.InsertBookRow(t, app.db, "Dune", nil)
	body := bytes.NewBufferString(`{
		"started_at":"2026-01-01",
		"finished_at":"2026-01-03",
		"rating":4.5,
		"notes":"Excellent"
	}`)

	req := httptest.NewRequest(http.MethodPost, "/api/books/"+bookID+"/reads", body)
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
	if response["book_id"] != bookID || response["rating"] != 4.5 || response["notes"] != "Excellent" {
		t.Fatalf("unexpected response: %#v", response)
	}
	if _, exists := response["created_at"]; exists {
		t.Fatal("created_at must not be exposed")
	}
	if _, exists := response["updated_at"]; exists {
		t.Fatal("updated_at must not be exposed")
	}

	var startedAt, finishedAt, notes sql.NullString
	var rating sql.NullFloat64
	var createdAt, updatedAt string

	err := app.db.QueryRowContext(context.Background(), `
		SELECT started_at, finished_at, rating, notes, created_at, updated_at
		FROM reads WHERE id = ?
	`, response["id"]).Scan(&startedAt, &finishedAt, &rating, &notes, &createdAt, &updatedAt)

	if err != nil {
		t.Fatal(err)
	}
	if startedAt.String != "2026-01-01" || finishedAt.String != "2026-01-03" || rating.Float64 != 4.5 || notes.String != "Excellent" {
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
	req := httptest.NewRequest(http.MethodPost, "/api/books/"+bookID+"/reads", body)
	resp := httptest.NewRecorder()

	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusBadRequest {
		t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, resp.Code, resp.Body.String())
	}
}

func TestReadAPICreateReturns404ForUnknownBook(t *testing.T) {
	app := newTestApp(t)
	req := httptest.NewRequest(http.MethodPost, "/api/books/"+uuid.NewString()+"/reads", bytes.NewBufferString(`{}`))
	resp := httptest.NewRecorder()

	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d: %s", http.StatusNotFound, resp.Code, resp.Body.String())
	}
}

func TestReadAPICreateRejectsInvalidValues(t *testing.T) {
	tests := []string{
		`{"started_at":"not-a-date"}`,
		`{"started_at":"2026-02-02","finished_at":"2026-02-01"}`,
		`{"rating":0.5}`,
		`{"rating":4.2}`,
		`{"rating":5.5}`,
	}
	for _, body := range tests {
		t.Run(body, func(t *testing.T) {
			app := newTestApp(t)
			bookID := testsupport.InsertBookRow(t, app.db, "Dune", nil)
			req := httptest.NewRequest(http.MethodPost, "/api/books/"+bookID+"/reads", bytes.NewBufferString(body))
			resp := httptest.NewRecorder()

			app.handler.ServeHTTP(resp, req)
			if resp.Code != http.StatusBadRequest {
				t.Fatalf("expected status %d, got %d: %s", http.StatusBadRequest, resp.Code, resp.Body.String())
			}
		})
	}
}

// ── List ──────────────────────────────────────────────────────────────────────

func TestReadAPIList(t *testing.T) {
	app := newTestApp(t)
	bookID := testsupport.InsertBookRow(t, app.db, "Dune", nil)
	testsupport.InsertReadRow(t, app.db, testsupport.ReadRow{
		ID: "older", BookID: bookID, StartedAt: new("2025-01-01"), FinishedAt: new("2025-01-03"),
		Rating: new(4.0), Notes: new("Good"), CreatedAt: "2026-01-01T00:00:00Z",
	})
	testsupport.InsertReadRow(t, app.db, testsupport.ReadRow{
		ID: "newer", BookID: bookID, StartedAt: new("2026-01-01"), FinishedAt: new("2026-01-03"),
		Rating: new(4.5), Notes: new("Excellent"), CreatedAt: "2026-01-02T00:00:00Z",
	})
	req := httptest.NewRequest(http.MethodGet, "/api/books/"+bookID+"/reads", nil)
	resp := httptest.NewRecorder()

	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}

	var raw []map[string]any
	if err := json.NewDecoder(resp.Body).Decode(&raw); err != nil {
		t.Fatal(err)
	}
	if len(raw) != 2 || raw[0]["id"] != "newer" || raw[1]["id"] != "older" {
		t.Fatalf("expected newest reads first, got %#v", raw)
	}
	for _, item := range raw {
		for _, field := range []string{"id", "started_at", "finished_at", "rating", "notes"} {
			if _, exists := item[field]; !exists {
				t.Fatalf("expected field %q in %#v", field, item)
			}
		}
		if _, exists := item["book_id"]; exists {
			t.Fatal("book_id must not be exposed by the scoped list endpoint")
		}
		if _, exists := item["created_at"]; exists {
			t.Fatal("created_at must not be exposed")
		}
		if _, exists := item["updated_at"]; exists {
			t.Fatal("updated_at must not be exposed")
		}
	}
}

func TestReadAPIListReturnsEmptyArray(t *testing.T) {
	app := newTestApp(t)
	bookID := testsupport.InsertBookRow(t, app.db, "Dune", nil)
	req := httptest.NewRequest(http.MethodGet, "/api/books/"+bookID+"/reads", nil)
	resp := httptest.NewRecorder()

	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK || resp.Body.String() != "[]\n" {
		t.Fatalf("expected empty JSON array, got status %d: %s", resp.Code, resp.Body.String())
	}
}

func TestReadAPIListReturns404ForUnknownBook(t *testing.T) {
	app := newTestApp(t)
	req := httptest.NewRequest(http.MethodGet, "/api/books/"+uuid.NewString()+"/reads", nil)
	resp := httptest.NewRecorder()

	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, resp.Code)
	}
}
