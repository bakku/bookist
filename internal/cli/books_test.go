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

// ── Books List ─────────────────────────────────────────────────────────────────

func TestBooksListPrintsBooks(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode([]books.Book{
			{ID: "id-1", Title: "Dune", ISBN: new("9780441172719")},
			{ID: "id-2", Title: "Kindred", ISBN: nil},
		})
	}))
	defer server.Close()

	var stdout, stderr strings.Builder
	exitCode := cli.Run([]string{"books", "list", "--server", server.URL}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", exitCode, stderr.String())
	}

	output := strings.TrimSpace(stdout.String())
	if !strings.Contains(output, "id-1\tDune\t9780441172719") {
		t.Fatalf("expected output to contain book with ISBN, got %q", output)
	}
	if !strings.Contains(output, "id-2\tKindred") {
		t.Fatalf("expected output to contain book without ISBN, got %q", output)
	}
}

// ── Books Add ──────────────────────────────────────────────────────────────────

func TestBooksAddWithNewFields(t *testing.T) {
	var postedBooks []books.CreateBookRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/authors":
			json.NewEncoder(w).Encode([]authors.Author{})

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

	exitCode := cli.Run([]string{
		"books", "add", "--title", "Full Book", "--language", "en", "--publisher", "O'Reilly",
		"--edition", "2nd", "--format", "paperback", "--purchased-at", "2025-06-15",
		"--pages", "400", "--notes", "Great read", "--published-year", "2024",
		"--published-month", "6", "--published-day", "15", "--server", server.URL},
		&stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", exitCode, stderr.String())
	}

	if len(postedBooks) != 1 {
		t.Fatalf("expected 1 POST /api/books, got %d", len(postedBooks))
	}

	got := postedBooks[0]

	if got.Title != "Full Book" {
		t.Fatalf("expected title 'Full Book', got %q", got.Title)
	}

	if got.Language == nil || *got.Language != "en" {
		t.Fatalf("expected language 'en', got %#v", got.Language)
	}

	if got.Publisher == nil || *got.Publisher != "O'Reilly" {
		t.Fatalf("expected publisher 'O\\'Reilly', got %#v", got.Publisher)
	}

	if got.Edition == nil || *got.Edition != "2nd" {
		t.Fatalf("expected edition '2nd', got %#v", got.Edition)
	}

	if got.Format == nil || *got.Format != books.FormatPaperback {
		t.Fatalf("expected format paperback, got %#v", got.Format)
	}

	if got.PurchasedAt == nil || *got.PurchasedAt != "2025-06-15" {
		t.Fatalf("expected purchased_at '2025-06-15', got %#v", got.PurchasedAt)
	}

	if got.Pages == nil || *got.Pages != 400 {
		t.Fatalf("expected pages 400, got %#v", got.Pages)
	}

	if got.Notes == nil || *got.Notes != "Great read" {
		t.Fatalf("expected notes 'Great read', got %#v", got.Notes)
	}

	if got.PublishedYear == nil || *got.PublishedYear != 2024 {
		t.Fatalf("expected published_year 2024, got %#v", got.PublishedYear)
	}

	if got.PublishedMonth == nil || *got.PublishedMonth != 6 {
		t.Fatalf("expected published_month 6, got %#v", got.PublishedMonth)
	}

	if got.PublishedDay == nil || *got.PublishedDay != 15 {
		t.Fatalf("expected published_day 15, got %#v", got.PublishedDay)
	}
}

func TestBooksAddSendsNullForOmittedOptionalFields(t *testing.T) {
	var posted books.CreateBookRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&posted); err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(books.Book{ID: "new-book-id", Title: posted.Title})
	}))
	defer server.Close()

	var stdout, stderr strings.Builder
	exitCode := cli.Run([]string{"books", "add", "--title", "Minimal", "--server", server.URL}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", exitCode, stderr.String())
	}
	if posted.ISBN != nil || posted.Language != nil || posted.Publisher != nil || posted.Edition != nil ||
		posted.Format != nil || posted.PurchasedAt != nil || posted.Pages != nil || posted.Notes != nil ||
		posted.PublishedYear != nil || posted.PublishedMonth != nil || posted.PublishedDay != nil {
		t.Fatalf("expected omitted optional fields to be nil, got %#v", posted)
	}
}

func TestBooksAddRejectsInvalidIntegerFlag(t *testing.T) {
	var stdout, stderr strings.Builder
	exitCode := cli.Run([]string{"books", "add", "--title", "Invalid", "--pages", "many"}, &stdout, &stderr)

	if exitCode != 2 {
		t.Fatalf("expected exit code 2, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "invalid value \"many\" for flag -pages") {
		t.Fatalf("expected invalid pages error, got %q", stderr.String())
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
