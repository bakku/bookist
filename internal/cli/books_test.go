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

func TestBooksListTableFormats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]books.Book{
			{ID: 1, Title: "Dune", ISBN: new("9780441172719")},
			{ID: 2, Title: "Kindred", ISBN: nil},
		})
	}))
	defer server.Close()

	tests := []struct {
		name     string
		format   string
		expected string
	}{
		{name: "default pretty", expected: "ID  TITLE    ISBN\n1   Dune     9780441172719\n2   Kindred\n"},
		{name: "explicit TSV", format: "tsv", expected: "1\tDune\t9780441172719\n2\tKindred\t\n"},
		{name: "pretty", format: "pretty", expected: "ID  TITLE    ISBN\n1   Dune     9780441172719\n2   Kindred\n"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := []string{"books", "list", "--server", server.URL}
			if test.format != "" {
				args = append(args, "--format", test.format)
			}

			exitCode, stdout, stderr := runCLI(args)
			if exitCode != 0 {
				t.Fatalf("expected exit code 0, got %d; stderr: %s", exitCode, stderr)
			}
			if stderr != "" {
				t.Fatalf("expected empty stderr, got %q", stderr)
			}
			if stdout != test.expected {
				t.Fatalf("expected stdout %q, got %q", test.expected, stdout)
			}
		})
	}
}

func TestBooksListJSONPreservesNullableFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]books.Book{
			{
				ID:                1,
				Title:             "Dune",
				ISBN:              new("9780441172719"),
				Authors:           []authors.Author{{ID: 10, Name: "Frank Herbert"}},
				Summary:           new("A desert epic"),
				SeriesName:        new("Dune"),
				SeriesPosition:    new(1.5),
				Location:          new("Living room"),
				Condition:         new(books.ConditionVeryGood),
				AcquisitionSource: new("Bookshop"),
			},
			{ID: 2, Title: "Kindred", ISBN: nil},
		})
	}))
	defer server.Close()

	exitCode, stdout, stderr := runCLI([]string{"books", "list", "--server", server.URL, "--format", "json"})
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var listed []books.Book
	if err := json.Unmarshal([]byte(stdout), &listed); err != nil {
		t.Fatalf("expected valid JSON, got %q: %v", stdout, err)
	}
	if len(listed) != 2 {
		t.Fatalf("expected 2 books, got %d", len(listed))
	}
	if listed[0].ISBN == nil || *listed[0].ISBN != "9780441172719" {
		t.Fatalf("expected populated ISBN, got %#v", listed[0].ISBN)
	}
	if len(listed[0].Authors) != 1 || listed[0].Authors[0].Name != "Frank Herbert" {
		t.Fatalf("expected complete book JSON with authors, got %#v", listed[0])
	}
	if listed[0].Summary == nil || *listed[0].Summary != "A desert epic" ||
		listed[0].SeriesPosition == nil || *listed[0].SeriesPosition != 1.5 ||
		listed[0].Condition == nil || *listed[0].Condition != books.ConditionVeryGood {
		t.Fatalf("expected extended metadata in JSON, got %#v", listed[0])
	}
	if listed[1].ISBN != nil || listed[1].Publisher != nil || listed[1].Summary != nil ||
		listed[1].SeriesName != nil || listed[1].SeriesPosition != nil || listed[1].Location != nil ||
		listed[1].Condition != nil || listed[1].AcquisitionSource != nil {
		t.Fatalf("expected nullable fields to remain nil, got %#v", listed[1])
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
			json.NewEncoder(w).Encode(books.Book{ID: 10, Title: req.Title})
		}
	}))
	defer server.Close()

	var stdout, stderr strings.Builder

	exitCode := cli.Run([]string{
		"books", "add", "--title", "Full Book", "--language", "en", "--publisher", "O'Reilly",
		"--edition", "2nd", "--format", "paperback", "--purchased-at", "2025-06-15",
		"--pages", "400", "--notes", "Great read", "--summary", "A practical guide",
		"--series-name", "Programming", "--series-position", "1.5", "--location", "Office shelf",
		"--condition", "very_good", "--acquisition-source", "Bookshop", "--published-year", "2024",
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

	if got.Summary == nil || *got.Summary != "A practical guide" {
		t.Fatalf("expected summary, got %#v", got.Summary)
	}

	if got.SeriesName == nil || *got.SeriesName != "Programming" || got.SeriesPosition == nil || *got.SeriesPosition != 1.5 {
		t.Fatalf("expected series metadata, got %#v", got)
	}

	if got.Location == nil || *got.Location != "Office shelf" {
		t.Fatalf("expected location, got %#v", got.Location)
	}

	if got.Condition == nil || *got.Condition != books.ConditionVeryGood {
		t.Fatalf("expected condition very_good, got %#v", got.Condition)
	}

	if got.AcquisitionSource == nil || *got.AcquisitionSource != "Bookshop" {
		t.Fatalf("expected acquisition source, got %#v", got.AcquisitionSource)
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
		_ = json.NewEncoder(w).Encode(books.Book{ID: 10, Title: posted.Title})
	}))
	defer server.Close()

	var stdout, stderr strings.Builder
	exitCode := cli.Run([]string{"books", "add", "--title", "Minimal", "--server", server.URL}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", exitCode, stderr.String())
	}
	if posted.ISBN != nil || posted.Language != nil || posted.Publisher != nil || posted.Edition != nil ||
		posted.Format != nil || posted.PurchasedAt != nil || posted.Pages != nil || posted.Notes != nil ||
		posted.Summary != nil || posted.SeriesName != nil || posted.SeriesPosition != nil ||
		posted.Location != nil || posted.Condition != nil || posted.AcquisitionSource != nil ||
		posted.PublishedYear != nil || posted.PublishedMonth != nil || posted.PublishedDay != nil {
		t.Fatalf("expected omitted optional fields to be nil, got %#v", posted)
	}
}

func TestBooksAddRejectsInvalidFloatFlag(t *testing.T) {
	var stdout, stderr strings.Builder
	exitCode := cli.Run([]string{"books", "add", "--title", "Invalid", "--series-position", "first"}, &stdout, &stderr)

	if exitCode != 2 {
		t.Fatalf("expected exit code 2, got %d", exitCode)
	}
	if !strings.Contains(stderr.String(), "invalid value \"first\" for flag -series-position") {
		t.Fatalf("expected invalid series position error, got %q", stderr.String())
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
					{ID: 5, Name: "Existing Author"},
				})

			case http.MethodPost:
				t.Fatal("unexpected POST /api/authors")
			}

		case "/api/books":
			var req books.CreateBookRequest
			json.NewDecoder(r.Body).Decode(&req)
			postedBooks = append(postedBooks, req)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(books.Book{ID: 10, Title: req.Title})
		}
	}))
	defer server.Close()

	var stdout, stderr strings.Builder
	exitCode := cli.Run([]string{"books", "add", "--title", "My Book", "--author", "existing author", "--server", server.URL}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", exitCode, stderr.String())
	}
	if len(postedBooks) != 1 {
		t.Fatalf("expected 1 POST /api/books, got %d", len(postedBooks))
	}
	if len(postedBooks[0].AuthorIDs) != 1 || postedBooks[0].AuthorIDs[0] != 5 {
		t.Fatalf("expected author_ids [5], got %#v", postedBooks[0].AuthorIDs)
	}
}

func TestBooksAddWithAmbiguousAuthorNameRequiresID(t *testing.T) {
	bookPosted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/authors":
			_ = json.NewEncoder(w).Encode([]authors.Author{
				{ID: 1, Name: "Alex Smith"},
				{ID: 2, Name: "alex SMITH"},
			})
		case "/api/books":
			bookPosted = true
		}
	}))
	defer server.Close()

	var stdout, stderr strings.Builder
	exitCode := cli.Run([]string{"books", "add", "--title", "My Book", "--author", "Alex Smith", "--server", server.URL}, &stdout, &stderr)

	if exitCode == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if bookPosted {
		t.Fatal("expected ambiguous author to prevent book creation")
	}
	if !strings.Contains(stderr.String(), `author "Alex Smith" exists multiple times; pass an author ID instead`) {
		t.Fatalf("unexpected error: %q", stderr.String())
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
				json.NewEncoder(w).Encode(authors.Author{ID: 5, Name: req.Name})
			}

		case "/api/books":
			var req books.CreateBookRequest
			json.NewDecoder(r.Body).Decode(&req)
			postedBooks = append(postedBooks, req)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(books.Book{ID: 10, Title: req.Title})
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
	if len(postedBooks[0].AuthorIDs) != 1 || postedBooks[0].AuthorIDs[0] != 5 {
		t.Fatalf("expected author_ids [5], got %#v", postedBooks[0].AuthorIDs)
	}
	if !strings.Contains(stdout.String(), "10\tMy Book") {
		t.Fatalf("expected stdout to contain '10\\tMy Book', got %q", stdout.String())
	}
}

func TestBooksAddWithIntegerAuthorIDLinksAuthor(t *testing.T) {
	var postedBooks []books.CreateBookRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/books":
			var req books.CreateBookRequest
			json.NewDecoder(r.Body).Decode(&req)
			postedBooks = append(postedBooks, req)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(books.Book{ID: 10, Title: req.Title})
		}
	}))
	defer server.Close()

	var stdout, stderr strings.Builder
	exitCode := cli.Run([]string{"books", "add", "--title", "My Book", "--author", "5", "--server", server.URL}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", exitCode, stderr.String())
	}
	if len(postedBooks) != 1 {
		t.Fatalf("expected 1 POST /api/books, got %d", len(postedBooks))
	}
	if len(postedBooks[0].AuthorIDs) != 1 || postedBooks[0].AuthorIDs[0] != 5 {
		t.Fatalf("expected author_ids [5], got %#v", postedBooks[0].AuthorIDs)
	}
}

func TestBooksAddPassesAuthorIntegerIDThroughWithoutLookup(t *testing.T) {
	var posted books.CreateBookRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/authors":
			t.Fatal("unexpected author lookup")

		case "/api/books":
			_ = json.NewDecoder(r.Body).Decode(&posted)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(books.Book{ID: 10, Title: posted.Title})
		}
	}))
	defer server.Close()

	var stdout, stderr strings.Builder
	exitCode := cli.Run([]string{"books", "add", "--title", "My Book", "--author", "5", "--server", server.URL}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("expected success, got %d: %s", exitCode, stderr.String())
	}
	if len(posted.AuthorIDs) != 1 || posted.AuthorIDs[0] != 5 {
		t.Fatalf("expected integer ID to pass through, got %#v", posted.AuthorIDs)
	}
}

func TestBooksAddRejectsInvalidAuthorIntegerIDs(t *testing.T) {
	for _, id := range []string{"0", "999999999999999999999999"} {
		t.Run(id, func(t *testing.T) {
			var stdout, stderr strings.Builder
			exitCode := cli.Run([]string{"books", "add", "--title", "My Book", "--author", id}, &stdout, &stderr)

			if exitCode == 0 {
				t.Fatal("expected non-zero exit code")
			}
			if !strings.Contains(stderr.String(), `invalid ID "`+id+`"`) {
				t.Fatalf("unexpected error: %q", stderr.String())
			}
		})
	}
}
