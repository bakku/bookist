package cli_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/cli"
)

func TestBooksAddWithAuthorNameNotFoundCreatesAuthorThenBook(t *testing.T) {
	var postedAuthors []authors.CreateAuthorRequest
	var postedBooks []books.CreateBookRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/authors":
			switch r.Method {
			case http.MethodGet:
				json.NewEncoder(w).Encode([]authors.Author{})
			case http.MethodPost:
				var req authors.CreateAuthorRequest
				json.NewDecoder(r.Body).Decode(&req)
				postedAuthors = append(postedAuthors, req)
				w.WriteHeader(http.StatusCreated)
				json.NewEncoder(w).Encode(authors.Author{ID: "new-author-id", Name: req.Name})
			}
		case "/api/books":
			var req books.CreateBookRequest
			json.NewDecoder(r.Body).Decode(&req)
			postedBooks = append(postedBooks, req)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(books.Book{ID: "new-book-id", Title: req.Title})
		}
	}))
	defer server.Close()

	var stdout, stderr strings.Builder
	exitCode := cli.Run([]string{"books", "add", "--title", "My Book", "--author", "New Author", "--server", server.URL}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", exitCode, stderr.String())
	}
	if len(postedAuthors) != 1 || postedAuthors[0].Name != "New Author" {
		t.Fatalf("expected POST /api/authors with 'New Author', got %#v", postedAuthors)
	}
	if len(postedBooks) != 1 {
		t.Fatalf("expected 1 POST /api/books, got %d", len(postedBooks))
	}
	if len(postedBooks[0].AuthorIDs) != 1 || postedBooks[0].AuthorIDs[0] != "new-author-id" {
		t.Fatalf("expected author_ids [new-author-id], got %#v", postedBooks[0].AuthorIDs)
	}
	if !strings.Contains(stdout.String(), "new-book-id\tMy Book") {
		t.Fatalf("expected stdout to contain 'new-book-id\\tMy Book', got %q", stdout.String())
	}
}

func TestBooksAddWithAuthorNameExistsLinksAuthor(t *testing.T) {
	var postedBooks []books.CreateBookRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/authors":
			switch r.Method {
			case http.MethodGet:
				json.NewEncoder(w).Encode([]authors.Author{
					{ID: "existing-id", Name: "Existing Author"},
				})
			case http.MethodPost:
				t.Fatal("unexpected POST /api/authors")
			}
		case "/api/books":
			var req books.CreateBookRequest
			json.NewDecoder(r.Body).Decode(&req)
			postedBooks = append(postedBooks, req)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(books.Book{ID: "new-book-id", Title: req.Title})
		}
	}))
	defer server.Close()

	var stdout, stderr strings.Builder
	exitCode := cli.Run([]string{"books", "add", "--title", "My Book", "--author", "Existing Author", "--server", server.URL}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", exitCode, stderr.String())
	}
	if len(postedBooks) != 1 {
		t.Fatalf("expected 1 POST /api/books, got %d", len(postedBooks))
	}
	if len(postedBooks[0].AuthorIDs) != 1 || postedBooks[0].AuthorIDs[0] != "existing-id" {
		t.Fatalf("expected author_ids [existing-id], got %#v", postedBooks[0].AuthorIDs)
	}
}

func TestBooksAddWithAuthorUUIDExistsLinksAuthor(t *testing.T) {
	var postedBooks []books.CreateBookRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/authors":
			json.NewEncoder(w).Encode([]authors.Author{
				{ID: "550e8400-e29b-41d4-a716-446655440000", Name: "UUID Author"},
			})
		case "/api/books":
			var req books.CreateBookRequest
			json.NewDecoder(r.Body).Decode(&req)
			postedBooks = append(postedBooks, req)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(books.Book{ID: "new-book-id", Title: req.Title})
		}
	}))
	defer server.Close()

	var stdout, stderr strings.Builder
	exitCode := cli.Run([]string{"books", "add", "--title", "My Book", "--author", "550e8400-e29b-41d4-a716-446655440000", "--server", server.URL}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", exitCode, stderr.String())
	}
	if len(postedBooks) != 1 {
		t.Fatalf("expected 1 POST /api/books, got %d", len(postedBooks))
	}
	if len(postedBooks[0].AuthorIDs) != 1 || postedBooks[0].AuthorIDs[0] != "550e8400-e29b-41d4-a716-446655440000" {
		t.Fatalf("expected author_ids [550e8400...], got %#v", postedBooks[0].AuthorIDs)
	}
}

func TestBooksAddWithAuthorUUIDNotFoundExitsNonZero(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/authors":
			json.NewEncoder(w).Encode([]authors.Author{})
		case "/api/books":
			t.Fatal("unexpected POST /api/books")
		}
	}))
	defer server.Close()

	var stdout, stderr strings.Builder
	exitCode := cli.Run([]string{"books", "add", "--title", "My Book", "--author", "550e8400-e29b-41d4-a716-446655440000", "--server", server.URL}, &stdout, &stderr)

	if exitCode == 0 {
		t.Fatalf("expected non-zero exit code, got 0")
	}
	if !strings.Contains(stderr.String(), "author not found") {
		t.Fatalf("expected stderr to contain 'author not found', got %q", stderr.String())
	}
}
