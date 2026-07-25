package httpserver_test

import (
	"bytes"
	"context"
	"database/sql"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/covers"
	"bakku.dev/bookist/internal/httpserver"
	"bakku.dev/bookist/internal/lists"
	"bakku.dev/bookist/internal/reads"
	"bakku.dev/bookist/internal/testsupport"
)

func assertSQLCount(t *testing.T, db *sql.DB, want int, query string, args ...any) {
	t.Helper()

	var got int
	if err := db.QueryRowContext(context.Background(), query, args...).Scan(&got); err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("expected count %d, got %d", want, got)
	}
}

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

func newTestApp(t *testing.T) testApp {
	t.Helper()

	db := testsupport.OpenMigratedDB(t)

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

	readRepo := reads.NewSQLiteRepository(db)
	readService := reads.NewService(readRepo)

	server, err := httpserver.New(bookService, authorService, listService, readService, coverStore)
	if err != nil {
		t.Fatal(err)
	}

	return testApp{
		handler:  server.Handler(),
		db:       db,
		coverDir: coverDir,
	}
}

// ── Book Covers ───────────────────────────────────────────────────────────────

func TestBookCoverRouteRejectsUnmanagedNames(t *testing.T) {
	app := newTestApp(t)
	req := httptest.NewRequest(http.MethodGet, "/book-covers/not-managed.png", nil)
	resp := httptest.NewRecorder()
	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, resp.Code)
	}
}

func TestBookCoverRouteServesManagedFile(t *testing.T) {
	app := newTestApp(t)
	key := "0123456789abcdef0123456789abcdef.png"
	image := []byte("\x89PNG\r\n\x1a\ncover")
	if err := os.WriteFile(filepath.Join(app.coverDir, key), image, 0o600); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/book-covers/"+key, nil)
	resp := httptest.NewRecorder()
	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}
	if got := resp.Header().Get("Content-Type"); got != "image/png" {
		t.Fatalf("expected image/png, got %q", got)
	}
	served, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(served, image) {
		t.Fatalf("expected served bytes %v, got %v", image, served)
	}
}
