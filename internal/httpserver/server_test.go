package httpserver_test

import (
	"database/sql"
	"net/http"
	"testing"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/httpserver"
	"bakku.dev/bookist/internal/lists"
	"bakku.dev/bookist/internal/testsupport"
)

type testApp struct {
	handler http.Handler
	db      *sql.DB
}

func newTestApp(t *testing.T) testApp {
	t.Helper()

	db := testsupport.OpenMigratedDB(t)

	authorRepo := authors.NewSQLiteRepository(db)
	authorService := authors.NewService(authorRepo)

	listRepo := lists.NewSQLiteRepository(db)
	listService := lists.NewService(listRepo)

	bookRepo := books.NewSQLiteRepository(db)
	bookService := books.NewService(bookRepo, authorRepo)

	server, err := httpserver.New(bookService, authorService, listService)
	if err != nil {
		t.Fatal(err)
	}

	return testApp{
		handler: server.Handler(),
		db:      db,
	}
}
