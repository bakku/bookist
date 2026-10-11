package api_test

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/covers"
	"bakku.dev/bookist/internal/httpserver/api"
	"bakku.dev/bookist/internal/lists"
	"bakku.dev/bookist/internal/reads"
	"bakku.dev/bookist/internal/testsupport"
)

type testApp struct {
	handler  http.Handler
	db       *sql.DB
	coverDir string
}

func patchJSON(t *testing.T, handler http.Handler, path, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPatch, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	return resp
}

func postBody(t *testing.T, handler http.Handler, path, contentType, body string) *httptest.ResponseRecorder {
	t.Helper()

	req := httptest.NewRequest(http.MethodPost, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", contentType)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)

	return resp
}

// assertAPIError checks the status, content type and structured error body of
// a response. An empty field means the error must not name a field.
func assertAPIError(t *testing.T, resp *httptest.ResponseRecorder, status int, code, field string) {
	t.Helper()

	if resp.Code != status {
		t.Fatalf("expected status %d, got %d: %s", status, resp.Code, resp.Body.String())
	}
	if contentType := resp.Header().Get("Content-Type"); contentType != "application/json" {
		t.Fatalf("expected JSON content type, got %q", contentType)
	}
	var body struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			Field   string `json:"field"`
		} `json:"error"`
	}
	if err := json.Unmarshal(resp.Body.Bytes(), &body); err != nil {
		t.Fatalf("expected structured error, got %s: %v", resp.Body.String(), err)
	}
	if body.Error.Code != code || body.Error.Field != field || body.Error.Message == "" {
		t.Fatalf("expected error code %q field %q with a message, got %s", code, field, resp.Body.String())
	}
}

func assertEmptyJSONArray(t *testing.T, handler http.Handler, path string) {
	t.Helper()

	req := httptest.NewRequest(http.MethodGet, path, nil)
	resp := httptest.NewRecorder()
	handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d: %s", http.StatusOK, resp.Code, resp.Body.String())
	}
	if body := strings.TrimSpace(resp.Body.String()); body != "[]" {
		t.Fatalf("expected empty JSON array, got %s", body)
	}
}

func newTestApp(t *testing.T) testApp {
	t.Helper()

	db := testsupport.OpenMigratedDB(t)
	return newTestAppWithReadRepository(t, db, reads.NewSQLiteRepository(db))
}

func newTestAppWithReadRepository(t *testing.T, db *sql.DB, readRepo reads.Repository) testApp {
	t.Helper()

	authorRepo := authors.NewSQLiteRepository(db)
	authorService := authors.NewService(authorRepo)

	listRepo := lists.NewSQLiteRepository(db)
	listService := lists.NewService(listRepo)

	bookRepo := books.NewSQLiteRepository(db)
	coverDir := t.TempDir()
	coverStore, err := covers.NewStore(coverDir)
	if err != nil {
		t.Fatal(err)
	}
	bookService := books.NewService(bookRepo, authorRepo, coverStore)

	readService := reads.NewService(readRepo)

	mux := http.NewServeMux()
	api.New(bookService, authorService, listService, readService).RegisterRoutes(mux)

	return testApp{
		handler:  mux,
		db:       db,
		coverDir: coverDir,
	}
}

// ── Unknown Routes ────────────────────────────────────────────────────────────

func TestAPIUnknownPathReturnsJSONNotFound(t *testing.T) {
	app := newTestApp(t)

	for _, path := range []string{"/api/nope", "/api/books/", "/api/books/1/nope"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			resp := httptest.NewRecorder()
			app.handler.ServeHTTP(resp, req)

			assertAPIError(t, resp, http.StatusNotFound, "not_found", "")
		})
	}
}

func TestAPIUnsupportedMethodReturnsJSONMethodNotAllowed(t *testing.T) {
	app := newTestApp(t)

	req := httptest.NewRequest(http.MethodPatch, "/api/books", nil)
	resp := httptest.NewRecorder()
	app.handler.ServeHTTP(resp, req)

	assertAPIError(t, resp, http.StatusMethodNotAllowed, "method_not_allowed", "")
	if allow := resp.Header().Get("Allow"); allow != "GET, POST" {
		t.Fatalf("expected Allow GET, POST, got %q", allow)
	}
}

// ── POST Bodies ───────────────────────────────────────────────────────────────

type postRoute struct {
	name   string
	path   string
	body   string
	count  string
	seeded int
}

// seedPostRoutes seeds the rows the POST routes need and returns each route
// with a valid body and the row count that must remain after a rejected request.
func seedPostRoutes(t *testing.T, app testApp) []postRoute {
	t.Helper()

	bookID := testsupport.InsertBookRow(t, app.db, "Dune", nil)
	listID := testsupport.InsertListRow(t, app.db, "Want to Buy")

	return []postRoute{
		{name: "books", path: "/api/books", body: `{"title":"Emma"}`, count: `SELECT COUNT(*) FROM books`, seeded: 1},
		{name: "authors", path: "/api/authors", body: `{"name":"Jane Austen"}`, count: `SELECT COUNT(*) FROM authors`},
		{name: "lists", path: "/api/lists", body: `{"name":"Nightstand"}`, count: `SELECT COUNT(*) FROM lists`, seeded: 1},
		{name: "list books", path: fmt.Sprintf("/api/lists/%d/books", listID), body: fmt.Sprintf(`{"book_id":%d}`, bookID), count: `SELECT COUNT(*) FROM book_lists`},
		{name: "reads", path: fmt.Sprintf("/api/books/%d/reads", bookID), body: `{}`, count: `SELECT COUNT(*) FROM reads`},
	}
}

func TestPostRejectsNonJSONContentType(t *testing.T) {
	app := newTestApp(t)

	for _, route := range seedPostRoutes(t, app) {
		t.Run(route.name, func(t *testing.T) {
			resp := postBody(t, app.handler, route.path, "text/plain", route.body)
			assertAPIError(t, resp, http.StatusUnsupportedMediaType, "unsupported_media_type", "")
			testsupport.AssertSQLCount(t, app.db, route.seeded, route.count)
		})
	}
}

func TestPostRejectsTrailingData(t *testing.T) {
	app := newTestApp(t)

	for _, route := range seedPostRoutes(t, app) {
		t.Run(route.name, func(t *testing.T) {
			resp := postBody(t, app.handler, route.path, "application/json", route.body+` {"x":1}`)
			assertAPIError(t, resp, http.StatusBadRequest, "invalid_json", "")
			testsupport.AssertSQLCount(t, app.db, route.seeded, route.count)
		})
	}
}

func TestPostRejectsOversizedBody(t *testing.T) {
	app := newTestApp(t)

	for _, route := range seedPostRoutes(t, app) {
		limit := 1 << 20
		if route.name == "books" {
			limit = 15 << 20
		}

		t.Run(route.name, func(t *testing.T) {
			resp := postBody(t, app.handler, route.path, "application/json", strings.Repeat(" ", limit+1)+route.body)
			assertAPIError(t, resp, http.StatusRequestEntityTooLarge, "body_too_large", "")
			testsupport.AssertSQLCount(t, app.db, route.seeded, route.count)
		})
	}
}
