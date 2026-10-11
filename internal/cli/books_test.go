package cli_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/cli"
	"bakku.dev/bookist/internal/lists"
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
			args := []string{"books", "ls", "--server", server.URL}
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

	exitCode, stdout, stderr := runCLI([]string{"books", "ls", "--server", server.URL, "--format", "json"})
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

func TestBooksLSForwardsQuery(t *testing.T) {
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("q")
		_ = json.NewEncoder(w).Encode([]books.Book{})
	}))
	defer server.Close()

	exitCode, _, stderr := runCLI([]string{"books", "ls", "--query", "Lord & Rings", "--server", server.URL})
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", exitCode, stderr)
	}
	if gotQuery != "Lord & Rings" {
		t.Fatalf("expected query %q, got %q", "Lord & Rings", gotQuery)
	}
}

func TestBooksLSResolvesExactListAndForwardsQuery(t *testing.T) {
	var gotListQuery, gotBookQuery, gotBookPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/lists":
			gotListQuery = r.URL.Query().Get("q")
			_ = json.NewEncoder(w).Encode([]lists.List{
				{ID: 1, Name: "Old Nightstand"},
				{ID: 2, Name: "Nightstand"},
			})
		case "/api/lists/2/books":
			gotBookPath = r.URL.Path
			gotBookQuery = r.URL.Query().Get("q")
			_ = json.NewEncoder(w).Encode([]books.Book{})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	exitCode, _, stderr := runCLI([]string{
		"books", "ls", "--list", "nightstand", "--query", "Lord", "--server", server.URL,
	})
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", exitCode, stderr)
	}
	if gotListQuery != "nightstand" {
		t.Fatalf("expected list query %q, got %q", "nightstand", gotListQuery)
	}
	if gotBookPath != "/api/lists/2/books" || gotBookQuery != "Lord" {
		t.Fatalf("expected filtered list books request, got path %q query %q", gotBookPath, gotBookQuery)
	}
}

// ── Books Show ─────────────────────────────────────────────────────────────────

func TestBooksShowByID(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/books" {
			_ = json.NewEncoder(w).Encode([]books.Book{})
			return
		}
		gotPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(showTestBook())
	}))
	defer server.Close()

	tests := []struct {
		name     string
		format   string
		expected string
	}{
		{name: "default pretty", expected: showBookPretty},
		{name: "pretty", format: "pretty", expected: showBookPretty},
		{name: "TSV", format: "tsv", expected: showBookTSV},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := []string{"books", "show", "--server", server.URL, "12"}
			if test.format != "" {
				args = append(args, "--format", test.format)
			}

			exitCode, stdout, stderr := runCLI(args)
			if exitCode != 0 || stderr != "" {
				t.Fatalf("unexpected result: exit=%d stderr=%q", exitCode, stderr)
			}
			if stdout != test.expected {
				t.Fatalf("expected stdout %q, got %q", test.expected, stdout)
			}
			if gotPath != "/api/books/12" {
				t.Fatalf("expected GET /api/books/12, got %s", gotPath)
			}
		})
	}
}

func TestBooksShowJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/books" {
			_ = json.NewEncoder(w).Encode([]books.Book{})
			return
		}
		_ = json.NewEncoder(w).Encode(showTestBook())
	}))
	defer server.Close()

	exitCode, stdout, stderr := runCLI([]string{"books", "show", "12", "--format", "json", "--server", server.URL})
	if exitCode != 0 || stderr != "" {
		t.Fatalf("unexpected result: exit=%d stderr=%q", exitCode, stderr)
	}

	var got books.Book
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("expected valid JSON, got %q: %v", stdout, err)
	}
	if got.ID != 12 || got.Title != "Dune" || len(got.Authors) != 2 || got.CoverURL == nil || *got.CoverURL != "/book-covers/abc.jpg" {
		t.Fatalf("expected complete book JSON, got %#v", got)
	}
}

func TestBooksShowOmitsUnsetFields(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/books" {
			_ = json.NewEncoder(w).Encode([]books.Book{})
			return
		}
		_ = json.NewEncoder(w).Encode(books.Book{ID: 3, Title: "Kindred", Authors: []authors.Author{}, PublishedYear: new(1979)})
	}))
	defer server.Close()

	exitCode, stdout, stderr := runCLI([]string{"books", "show", "3", "--format", "tsv", "--server", server.URL})
	if exitCode != 0 || stderr != "" {
		t.Fatalf("unexpected result: exit=%d stderr=%q", exitCode, stderr)
	}
	for _, line := range []string{"ISBN\t\n", "Authors\t\n", "Pages\t\n", "Published\t1979\n"} {
		if !strings.Contains(stdout, line) {
			t.Fatalf("expected stdout to contain %q, got %q", line, stdout)
		}
	}
}

func TestBooksShowResolvesTitle(t *testing.T) {
	var gotQuery, gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/books" {
			gotQuery = r.URL.Query().Get("q")
			_ = json.NewEncoder(w).Encode([]books.Book{{ID: 2, Title: "Dune Messiah"}, {ID: 7, Title: "Dune"}})
			return
		}
		gotPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(books.Book{ID: 7, Title: "Dune", Authors: []authors.Author{}})
	}))
	defer server.Close()

	exitCode, stdout, stderr := runCLI([]string{"books", "show", "--server", server.URL, "dune"})
	if exitCode != 0 || stderr != "" {
		t.Fatalf("unexpected result: exit=%d stderr=%q", exitCode, stderr)
	}
	if gotQuery != "dune" || gotPath != "/api/books/7" {
		t.Fatalf("expected title lookup then GET /api/books/7, got query %q path %q", gotQuery, gotPath)
	}
	if !strings.Contains(stdout, "Title") || !strings.Contains(stdout, "Dune") {
		t.Fatalf("expected book output, got %q", stdout)
	}
}

func TestBooksShowReportsUnknownBook(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/books" {
			_ = json.NewEncoder(w).Encode([]books.Book{})
			return
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"book_not_found","message":"book not found"}}`))
	}))
	defer server.Close()

	tests := []struct {
		name     string
		ref      string
		expected string
	}{
		{name: "unknown ID", ref: "999", expected: "book not found: 999"},
		{name: "unknown title", ref: "Missing", expected: "book not found: Missing"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			exitCode, stdout, stderr := runCLI([]string{"books", "show", "--server", server.URL, test.ref})
			if exitCode != 1 || stdout != "" || !strings.Contains(stderr, test.expected) {
				t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
			}
		})
	}
}

func TestBooksShowRejectsAmbiguousTitle(t *testing.T) {
	getCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/books" {
			getCalled = true
		}
		_ = json.NewEncoder(w).Encode([]books.Book{{ID: 1, Title: "Dune"}, {ID: 2, Title: "dune"}})
	}))
	defer server.Close()

	exitCode, stdout, stderr := runCLI([]string{"books", "show", "--server", server.URL, "Dune"})
	if exitCode != 1 || stdout != "" || !strings.Contains(stderr, "pass a book ID instead") {
		t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if getCalled {
		t.Fatal("expected no book to be fetched for an ambiguous title")
	}
}

func TestBooksShowRequiresExactlyOneReference(t *testing.T) {
	for _, args := range [][]string{{"books", "show"}, {"books", "show", "1", "2"}} {
		exitCode, stdout, stderr := runCLI(args)
		if exitCode != 2 || stdout != "" || !strings.Contains(stderr, "books show requires exactly one title or ID") {
			t.Fatalf("unexpected result for %v: exit=%d stdout=%q stderr=%q", args, exitCode, stdout, stderr)
		}
	}
}

const showBookPretty = "ID                  12\n" +
	"Title               Dune\n" +
	"ISBN                9780441172719\n" +
	"Authors             Frank Herbert, Brian Herbert\n" +
	"Language            en\n" +
	"Publisher           Chilton\n" +
	"Edition             1st\n" +
	"Format              paperback\n" +
	"Purchased at        2025-06-15\n" +
	"Purchase price      12.34 EUR\n" +
	"Pages               412\n" +
	"Notes               Classic\n" +
	"Summary             A desert epic\n" +
	"Series name         Dune\n" +
	"Series position     1.5\n" +
	"Location            Living room\n" +
	"Condition           very_good\n" +
	"Acquisition source  Bookshop\n" +
	"Published           1965-08-01\n" +
	"Cover URL           /book-covers/abc.jpg\n" +
	"Created at          2026-01-02T03:04:05Z\n" +
	"Updated at          2026-01-03T03:04:05Z\n"

const showBookTSV = "ID\t12\n" +
	"Title\tDune\n" +
	"ISBN\t9780441172719\n" +
	"Authors\tFrank Herbert, Brian Herbert\n" +
	"Language\ten\n" +
	"Publisher\tChilton\n" +
	"Edition\t1st\n" +
	"Format\tpaperback\n" +
	"Purchased at\t2025-06-15\n" +
	"Purchase price\t12.34 EUR\n" +
	"Pages\t412\n" +
	"Notes\tClassic\n" +
	"Summary\tA desert epic\n" +
	"Series name\tDune\n" +
	"Series position\t1.5\n" +
	"Location\tLiving room\n" +
	"Condition\tvery_good\n" +
	"Acquisition source\tBookshop\n" +
	"Published\t1965-08-01\n" +
	"Cover URL\t/book-covers/abc.jpg\n" +
	"Created at\t2026-01-02T03:04:05Z\n" +
	"Updated at\t2026-01-03T03:04:05Z\n"

func showTestBook() books.Book {
	format := books.FormatPaperback
	condition := books.ConditionVeryGood

	return books.Book{
		ID:    12,
		Title: "Dune",
		ISBN:  new("9780441172719"),
		Authors: []authors.Author{
			{ID: 1, Name: "Frank Herbert"},
			{ID: 2, Name: "Brian Herbert"},
		},
		Language:          new("en"),
		Publisher:         new("Chilton"),
		Edition:           new("1st"),
		Format:            &format,
		PurchasedAt:       new("2025-06-15"),
		PurchasePrice:     new("12.34 EUR"),
		Pages:             new(412),
		Notes:             new("Classic"),
		Summary:           new("A desert epic"),
		SeriesName:        new("Dune"),
		SeriesPosition:    new(1.5),
		Location:          new("Living room"),
		Condition:         &condition,
		AcquisitionSource: new("Bookshop"),
		PublishedYear:     new(1965),
		PublishedMonth:    new(8),
		PublishedDay:      new(1),
		CoverURL:          new("/book-covers/abc.jpg"),
		CreatedAt:         time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC),
		UpdatedAt:         time.Date(2026, 1, 3, 3, 4, 5, 0, time.UTC),
	}
}

// ── Books Add ──────────────────────────────────────────────────────────────────

func TestBooksAddWithCompleteFields(t *testing.T) {
	var postedBooks []books.CreateBookRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/authors":
			_ = json.NewEncoder(w).Encode([]authors.Author{})

		case "/api/books":
			var req books.CreateBookRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			postedBooks = append(postedBooks, req)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(books.Book{ID: 10, Title: req.Title})
		}
	}))
	defer server.Close()

	var stdout, stderr strings.Builder

	exitCode := cli.Run([]string{
		"books", "add", "--title", "Full Book", "--isbn", "12-3456-789", "--language", "en",
		"--publisher", "O'Reilly", "--edition", "2nd", "--format", "paperback",
		"--purchased-at", "2025-06-15", "--purchase-price", "12.34 EUR", "--pages", "400",
		"--notes", "Great read", "--summary", "A practical guide", "--series-name", "Programming",
		"--series-position", "1.5", "--location", "Office shelf", "--condition", "very_good",
		"--acquisition-source", "Bookshop", "--published-year", "2024", "--published-month", "6",
		"--published-day", "15", "--server", server.URL},
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

	if got.ISBN == nil || *got.ISBN != "12-3456-789" {
		t.Fatalf("expected isbn '12-3456-789', got %q", *got.ISBN)
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

	if got.PurchasePrice == nil || *got.PurchasePrice != "12.34 EUR" {
		t.Fatalf("expected purchase_price '12.34 EUR', got %#v", got.PurchasePrice)
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
		posted.Format != nil || posted.PurchasedAt != nil || posted.PurchasePrice != nil || posted.Pages != nil || posted.Notes != nil ||
		posted.Summary != nil || posted.SeriesName != nil || posted.SeriesPosition != nil ||
		posted.Location != nil || posted.Condition != nil || posted.AcquisitionSource != nil ||
		posted.PublishedYear != nil || posted.PublishedMonth != nil || posted.PublishedDay != nil {
		t.Fatalf("expected omitted optional fields to be nil, got %#v", posted)
	}
}

func TestBooksAddLoadsCoverFromFile(t *testing.T) {
	image := []byte("\x89PNG\r\n\x1a\ncover")
	path := filepath.Join(t.TempDir(), "cover.png")
	if err := os.WriteFile(path, image, 0o600); err != nil {
		t.Fatal(err)
	}

	var posted books.CreateBookRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if err := json.NewDecoder(r.Body).Decode(&posted); err != nil {
			t.Fatal(err)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(books.Book{ID: 10, Title: posted.Title})
	}))

	defer server.Close()

	exitCode, _, stderr := runCLI([]string{"books", "add", "--title", "Dune", "--cover", path, "--server", server.URL})

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", exitCode, stderr)
	}

	if posted.Cover == nil || !bytes.Equal(*posted.Cover, image) {
		t.Fatalf("expected cover bytes %v, got %#v", image, posted.Cover)
	}
}

func TestBooksAddLoadsCoverFromURL(t *testing.T) {
	image := []byte("\x89PNG\r\n\x1a\nremote")
	coverServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write(image)
	}))
	defer coverServer.Close()

	var posted books.CreateBookRequest

	apiServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&posted)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(books.Book{ID: 10, Title: posted.Title})
	}))

	defer apiServer.Close()

	exitCode, _, stderr := runCLI([]string{"books", "add", "--title", "Dune", "--cover", coverServer.URL, "--server", apiServer.URL})

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", exitCode, stderr)
	}
	if posted.Cover == nil || !bytes.Equal(*posted.Cover, image) {
		t.Fatalf("expected cover bytes %v, got %#v", image, posted.Cover)
	}
}

func TestBooksAddDoesNotFollowCoverRedirects(t *testing.T) {
	targetCalled := false
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		targetCalled = true
		_, _ = w.Write([]byte("\x89PNG\r\n\x1a\ncover"))
	}))
	defer target.Close()

	redirect := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer redirect.Close()

	exitCode, _, stderr := runCLI([]string{"books", "add", "--title", "Dune", "--cover", redirect.URL})

	if exitCode == 0 {
		t.Fatal("expected redirect to fail")
	}
	if targetCalled {
		t.Fatal("expected redirect target not to be requested")
	}
	if !strings.Contains(stderr, "302 Found") {
		t.Fatalf("expected redirect status error, got %q", stderr)
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
				if got := r.URL.Query().Get("q"); got != "existing author" {
					t.Fatalf("expected author query %q, got %q", "existing author", got)
				}

				_ = json.NewEncoder(w).Encode([]authors.Author{
					{ID: 4, Name: "Former Existing Author"},
					{ID: 5, Name: "Existing Author"},
				})

			case http.MethodPost:
				t.Fatal("unexpected POST /api/authors")
			}

		case "/api/books":
			var req books.CreateBookRequest
			_ = json.NewDecoder(r.Body).Decode(&req)

			postedBooks = append(postedBooks, req)

			w.WriteHeader(http.StatusCreated)

			_ = json.NewEncoder(w).Encode(books.Book{ID: 10, Title: req.Title})
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

func TestBooksAddWithUnknownAuthorFailsWithoutCreatingAnything(t *testing.T) {
	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method+" "+r.URL.Path)
		if r.Method == http.MethodGet && r.URL.Path == "/api/authors" {
			_ = json.NewEncoder(w).Encode([]authors.Author{{ID: 4, Name: "Frank Herbert"}})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	exitCode, stdout, stderr := runCLI([]string{"books", "add", "--title", "Dune", "--author", "Frank Herbret", "--server", server.URL})

	if exitCode != 1 || stdout != "" || !strings.Contains(stderr, "author not found: Frank Herbret") {
		t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if len(methods) != 1 || methods[0] != "GET /api/authors" {
		t.Fatalf("expected only an author lookup, got %#v", methods)
	}
}

func TestBooksAddCreateAuthorsCreatesMissingAuthorsAndReportsThem(t *testing.T) {
	var postedAuthors []authors.CreateAuthorRequest
	var postedBooks []books.CreateBookRequest

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/authors":
			switch r.Method {
			case http.MethodGet:
				_ = json.NewEncoder(w).Encode([]authors.Author{{ID: 4, Name: "Existing Author"}})

			case http.MethodPost:
				var req authors.CreateAuthorRequest
				_ = json.NewDecoder(r.Body).Decode(&req)
				postedAuthors = append(postedAuthors, req)
				w.WriteHeader(http.StatusCreated)
				_ = json.NewEncoder(w).Encode(authors.Author{ID: 5, Name: req.Name})
			}

		case "/api/books":
			var req books.CreateBookRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			postedBooks = append(postedBooks, req)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(books.Book{ID: 10, Title: req.Title})
		}
	}))
	defer server.Close()

	exitCode, stdout, stderr := runCLI([]string{
		"books", "add", "--title", "My Book", "--author", "Existing Author", "--author", "New Author",
		"--create-authors", "--server", server.URL,
	})

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", exitCode, stderr)
	}
	if len(postedAuthors) != 1 || postedAuthors[0].Name != "New Author" {
		t.Fatalf("expected one POST /api/authors with 'New Author', got %#v", postedAuthors)
	}
	if len(postedBooks) != 1 {
		t.Fatalf("expected 1 POST /api/books, got %d", len(postedBooks))
	}
	if !reflect.DeepEqual(postedBooks[0].AuthorIDs, []int64{4, 5}) {
		t.Fatalf("expected author_ids [4 5], got %#v", postedBooks[0].AuthorIDs)
	}
	if stderr != "created author \"New Author\" (id 5)\n" {
		t.Fatalf("expected created author report on stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "10\tMy Book") {
		t.Fatalf("expected stdout to contain '10\\tMy Book', got %q", stdout)
	}
}

func TestBooksAddCreateAuthorsNeverCreatesForExplicitID(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			t.Errorf("unexpected %s %s", r.Method, r.URL.Path)
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	exitCode, stdout, stderr := runCLI([]string{"books", "add", "--title", "My Book", "--author", "id:99", "--create-authors", "--server", server.URL})

	if exitCode != 1 || stdout != "" || !strings.Contains(stderr, "author not found: id:99") {
		t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
}

func TestBooksAddRejectsBlankAuthor(t *testing.T) {
	exitCode, stdout, stderr := runCLI([]string{"books", "add", "--title", "My Book", "--author", " "})

	if exitCode != 2 || stdout != "" || !strings.Contains(stderr, "--author must not be blank") {
		t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
}

func TestBooksAddRejectsInvalidExplicitAuthorIDs(t *testing.T) {
	for _, ref := range []string{"id:0", "id:abc", "id:999999999999999999999999"} {
		t.Run(ref, func(t *testing.T) {
			exitCode, _, stderr := runCLI([]string{"books", "add", "--title", "My Book", "--author", ref})

			if exitCode == 0 {
				t.Fatal("expected non-zero exit code")
			}
			if !strings.Contains(stderr, `invalid ID "`+ref+`"`) {
				t.Fatalf("unexpected error: %q", stderr)
			}
		})
	}
}

// ── Books Remove ──────────────────────────────────────────────────────────────

func TestBooksRemoveDeletesByIDAndPrintsResult(t *testing.T) {
	var method, path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if answerLookup(w, r, "/api/books", 12, books.Book{ID: 12, Title: "Dune"}) {
			return
		}
		method, path = r.Method, r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	exitCode, stdout, stderr := runCLI([]string{"books", "rm", "--server", server.URL, "12"})
	if exitCode != 0 || stderr != "" || stdout != "removed book 12\n" {
		t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if method != http.MethodDelete || path != "/api/books/12" {
		t.Fatalf("expected DELETE /api/books/12, got %s %s", method, path)
	}
}

func TestBooksRemoveResolvesExactTitle(t *testing.T) {
	var deletedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			if r.URL.Query().Get("q") != "dUnE" {
				t.Fatalf("unexpected query %q", r.URL.Query().Get("q"))
			}
			_ = json.NewEncoder(w).Encode([]books.Book{{ID: 1, Title: "Dune Messiah"}, {ID: 2, Title: "Dune"}})
			return
		}
		deletedPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	exitCode, _, stderr := runCLI([]string{"books", "rm", "--server", server.URL, "dUnE"})
	if exitCode != 0 || stderr != "" || deletedPath != "/api/books/2" {
		t.Fatalf("unexpected result: exit=%d path=%q stderr=%q", exitCode, deletedPath, stderr)
	}
}

func TestBooksRemoveRejectsAmbiguousTitle(t *testing.T) {
	deleteCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deleteCalled = true
		}
		_ = json.NewEncoder(w).Encode([]books.Book{{ID: 1, Title: "Dune"}, {ID: 2, Title: "dune"}})
	}))
	defer server.Close()

	exitCode, _, stderr := runCLI([]string{"books", "rm", "--server", server.URL, "Dune"})
	if exitCode == 0 || !strings.Contains(stderr, `book "Dune" exists multiple times; pass a book ID instead`) {
		t.Fatalf("unexpected result: exit=%d stderr=%q", exitCode, stderr)
	}
	if deleteCalled {
		t.Fatal("expected ambiguity to prevent DELETE")
	}
}

func TestBooksRemoveRequiresNoContentResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if answerLookup(w, r, "/api/books", 1, books.Book{ID: 1, Title: "Dune"}) {
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	exitCode, stdout, stderr := runCLI([]string{"books", "rm", "--server", server.URL, "1"})
	if exitCode != 1 || stdout != "" || !strings.Contains(stderr, "remove book: server returned 200 OK") {
		t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
}

// ── Books Edit ───────────────────────────────────────────────────────────────

func TestBooksEditPatchesOnlyChanges(t *testing.T) {
	var method, path, contentType string
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if answerLookup(w, r, "/api/books", 12, books.Book{ID: 12, Title: "Dune"}) ||
			answerLookup(w, r, "/api/authors", 4, authors.Author{ID: 4, Name: "Frank Herbert"}) ||
			answerLookup(w, r, "/api/authors", 8, authors.Author{ID: 8, Name: "Brian Herbert"}) {
			return
		}
		method, path, contentType = r.Method, r.URL.Path, r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&body)
		_ = json.NewEncoder(w).Encode(books.Book{ID: 12, Title: "Dune Messiah"})
	}))
	defer server.Close()

	code, stdout, stderr := runCLI([]string{"books", "edit", "--title", "Dune Messiah", "--pages", "256", "--author", "4", "--author", "8", "--server", server.URL, "12"})
	if code != 0 || stderr != "" || stdout != "12\tDune Messiah\n" {
		t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if method != http.MethodPatch || path != "/api/books/12" || contentType != "application/json" {
		t.Fatalf("unexpected request: %s %s content-type=%q", method, path, contentType)
	}
	expected := `{"author_ids":[4,8],"pages":256,"title":"Dune Messiah"}`
	encoded, _ := json.Marshal(body)
	if string(encoded) != expected {
		t.Fatalf("expected body %s, got %s", expected, encoded)
	}
}

func TestBooksEditClearSendsNullAndResolvesTitle(t *testing.T) {
	var patched map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/books":
			_ = json.NewEncoder(w).Encode([]books.Book{{ID: 7, Title: "Kindred"}})
		case r.Method == http.MethodPatch && r.URL.Path == "/api/books/7":
			_ = json.NewDecoder(r.Body).Decode(&patched)
			_ = json.NewEncoder(w).Encode(books.Book{ID: 7, Title: "Kindred"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	code, _, stderr := runCLI([]string{"books", "edit", "--clear", "authors", "--clear", "cover", "--server", server.URL, "kindred"})
	if code != 0 || stderr != "" {
		t.Fatalf("unexpected result: exit=%d stderr=%q", code, stderr)
	}
	if len(patched) != 2 || patched["author_ids"] != nil || patched["cover"] != nil {
		t.Fatalf("expected only null author_ids and cover, got %#v", patched)
	}
}

func TestBooksEditDoesNotCreateMissingAuthors(t *testing.T) {
	var writes []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if answerLookup(w, r, "/api/books", 12, books.Book{ID: 12, Title: "Dune"}) ||
			answerLookup(w, r, "/api/authors", 0, nil) {
			return
		}
		writes = append(writes, r.Method+" "+r.URL.Path)
		http.NotFound(w, r)
	}))
	defer server.Close()

	code, stdout, stderr := runCLI([]string{"books", "edit", "--author", "Missing Author", "--server", server.URL, "12"})
	if code != 1 || stdout != "" || !strings.Contains(stderr, "author not found: Missing Author") {
		t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if len(writes) != 0 {
		t.Fatalf("expected no writes, got %#v", writes)
	}
}
