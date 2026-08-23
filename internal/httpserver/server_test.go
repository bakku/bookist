package httpserver_test

import (
	"database/sql"
	"net/http"
	"testing"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/covers"
	"bakku.dev/bookist/internal/httpserver"
	"bakku.dev/bookist/internal/lists"
	"bakku.dev/bookist/internal/reads"
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
