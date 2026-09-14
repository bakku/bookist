package api_test

import (
	"bytes"
	"database/sql"
	"net/http"
	"net/http/httptest"
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

	mux := http.NewServeMux()
	api.New(bookService, authorService, listService, readService).RegisterRoutes(mux)

	return testApp{
		handler:  mux,
		db:       db,
		coverDir: coverDir,
	}
}
