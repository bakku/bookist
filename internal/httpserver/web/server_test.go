package web_test

import (
	"database/sql"
	"net/http"
	"testing"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/covers"
	"bakku.dev/bookist/internal/httpserver/web"
	"bakku.dev/bookist/internal/testsupport"
)

type testApp struct {
	handler  http.Handler
	db       *sql.DB
	coverDir string
}

func newTestApp(t *testing.T) testApp {
	t.Helper()

	db := testsupport.OpenMigratedDB(t)
	authorRepo := authors.NewSQLiteRepository(db)
	bookRepo := books.NewSQLiteRepository(db)
	coverDir := t.TempDir()
	coverStore, err := covers.NewStore(coverDir)
	if err != nil {
		t.Fatal(err)
	}

	server, err := web.New(books.NewService(bookRepo, authorRepo, coverStore), coverStore)
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	server.RegisterRoutes(mux)

	return testApp{
		handler:  mux,
		db:       db,
		coverDir: coverDir,
	}
}
