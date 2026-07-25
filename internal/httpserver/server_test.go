package httpserver_test

import (
	"context"
	"database/sql"
	"net/http"
	"testing"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/books"
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

	readRepo := reads.NewSQLiteRepository(db)
	readService := reads.NewService(readRepo)

	server, err := httpserver.New(bookService, authorService, listService, readService)
	if err != nil {
		t.Fatal(err)
	}

	return testApp{
		handler: server.Handler(),
		db:      db,
	}
}
