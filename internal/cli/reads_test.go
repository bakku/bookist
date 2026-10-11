package cli_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/reads"
)

// ── Reads List ────────────────────────────────────────────────────────────────

func TestReadsListResolvesBookAndPrintsTableFormats(t *testing.T) {
	startedAt := "2026-01-01"
	abandonedAt := "2026-01-03"
	rating := 4.5
	notes := "Excellent"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/books":
			if got := r.URL.Query().Get("q"); got != "Dune" {
				t.Fatalf("expected book query %q, got %q", "Dune", got)
			}
			_ = json.NewEncoder(w).Encode([]books.Book{{ID: 1, Title: "Dune"}})
		case "/api/books/1/reads":
			_ = json.NewEncoder(w).Encode([]reads.Read{{
				ID: 10, BookID: 1, StartedAt: &startedAt,
				AbandonedAt: &abandonedAt, Rating: &rating, Notes: &notes,
			}})
		default:
			http.NotFound(w, r)
		}
	}))

	defer server.Close()

	tests := []struct {
		name     string
		format   string
		expected string
	}{
		{name: "default pretty", expected: "ID  STARTED_AT  FINISHED_AT  ABANDONED_AT  RATING  NOTES\n10  2026-01-01               2026-01-03    4.5     Excellent\n"},
		{name: "explicit TSV", format: "tsv", expected: "10\t2026-01-01\t\t2026-01-03\t4.5\tExcellent\n"},
		{name: "pretty", format: "pretty", expected: "ID  STARTED_AT  FINISHED_AT  ABANDONED_AT  RATING  NOTES\n10  2026-01-01               2026-01-03    4.5     Excellent\n"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := []string{"reads", "ls", "--book", "Dune", "--server", server.URL}
			if test.format != "" {
				args = append(args, "--format", test.format)
			}
			exitCode, stdout, stderr := runCLI(args)
			if exitCode != 0 || stderr != "" || stdout != test.expected {
				t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
			}
		})
	}
}

func TestReadsListJSONUsesFullReadRepresentation(t *testing.T) {
	const bookID = int64(1)
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if answerLookup(w, r, "/api/books", bookID, books.Book{ID: bookID, Title: "Dune"}) {
			return
		}
		_ = json.NewEncoder(w).Encode([]reads.Read{{ID: 10, BookID: bookID, CreatedAt: now, UpdatedAt: now}})
	}))

	defer server.Close()

	exitCode, stdout, stderr := runCLI([]string{"reads", "ls", "--book", "1", "--server", server.URL, "--format", "json"})

	if exitCode != 0 || stderr != "" {
		t.Fatalf("unexpected result: exit=%d stderr=%q", exitCode, stderr)
	}
	if !strings.Contains(stdout, `"started_at":null`) || !strings.Contains(stdout, `"abandoned_at":null`) || !strings.Contains(stdout, `"book_id":1`) ||
		!strings.Contains(stdout, `"created_at":"2026-01-02T03:04:05Z"`) || !strings.Contains(stdout, `"updated_at":"2026-01-02T03:04:05Z"`) {
		t.Fatalf("unexpected JSON output: %s", stdout)
	}
}

func TestReadsListRequiresBook(t *testing.T) {
	exitCode, stdout, stderr := runCLI([]string{"reads", "ls"})

	if exitCode != 2 || stdout != "" || !strings.Contains(stderr, "--book is required") {
		t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
}

// ── Reads Show ────────────────────────────────────────────────────────────────

func TestReadsShowByID(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewEncoder(w).Encode(showTestRead())
	}))
	defer server.Close()

	tests := []struct {
		name     string
		format   string
		expected string
	}{
		{name: "default pretty", expected: "ID            9\nBook ID       3\nStarted at    2026-03-01\nFinished at   2026-03-10\nAbandoned at\nRating        4.5\nNotes         Great\nCreated at    2026-03-01T00:00:00Z\nUpdated at    2026-03-10T00:00:00Z\n"},
		{name: "TSV", format: "tsv", expected: "ID\t9\nBook ID\t3\nStarted at\t2026-03-01\nFinished at\t2026-03-10\nAbandoned at\t\nRating\t4.5\nNotes\tGreat\nCreated at\t2026-03-01T00:00:00Z\nUpdated at\t2026-03-10T00:00:00Z\n"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := []string{"reads", "show", "9", "--server", server.URL}
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
			if gotPath != "/api/reads/9" {
				t.Fatalf("expected GET /api/reads/9, got %s", gotPath)
			}
		})
	}
}

func TestReadsShowJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(showTestRead())
	}))
	defer server.Close()

	exitCode, stdout, stderr := runCLI([]string{"reads", "show", "9", "--format", "json", "--server", server.URL})
	if exitCode != 0 || stderr != "" {
		t.Fatalf("unexpected result: exit=%d stderr=%q", exitCode, stderr)
	}

	var got reads.Read
	if err := json.Unmarshal([]byte(stdout), &got); err != nil {
		t.Fatalf("expected valid JSON, got %q: %v", stdout, err)
	}
	if got.ID != 9 || got.BookID != 3 || got.Rating == nil || *got.Rating != 4.5 || got.AbandonedAt != nil {
		t.Fatalf("expected complete read JSON, got %#v", got)
	}
}

func TestReadsShowReportsUnknownRead(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"error":{"code":"read_not_found","message":"read not found"}}`))
	}))
	defer server.Close()

	exitCode, stdout, stderr := runCLI([]string{"reads", "show", "--server", server.URL, "999"})
	if exitCode != 1 || stdout != "" || !strings.Contains(stderr, "show read: server returned 404 Not Found: read not found") {
		t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
}

func TestReadsShowRejectsInvalidReadID(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
	}))
	defer server.Close()

	tests := []struct {
		ref      string
		expected string
	}{
		{ref: "Dune", expected: `invalid read ID "Dune"`},
		{ref: "0", expected: `invalid ID "0"`},
	}
	for _, test := range tests {
		exitCode, stdout, stderr := runCLI([]string{"reads", "show", "--server", server.URL, test.ref})
		if exitCode != 2 || stdout != "" || !strings.Contains(stderr, test.expected) {
			t.Fatalf("unexpected result for %q: exit=%d stdout=%q stderr=%q", test.ref, exitCode, stdout, stderr)
		}
	}
	if requestCount != 0 {
		t.Fatalf("expected no HTTP requests, got %d", requestCount)
	}
}

func TestReadsShowRequiresExactlyOneReadID(t *testing.T) {
	for _, args := range [][]string{{"reads", "show"}, {"reads", "show", "1", "2"}} {
		exitCode, stdout, stderr := runCLI(args)
		if exitCode != 2 || stdout != "" || !strings.Contains(stderr, "reads show requires exactly one read ID") {
			t.Fatalf("unexpected result for %v: exit=%d stdout=%q stderr=%q", args, exitCode, stdout, stderr)
		}
	}
}

func showTestRead() reads.Read {
	return reads.Read{
		ID:         9,
		BookID:     3,
		StartedAt:  new("2026-03-01"),
		FinishedAt: new("2026-03-10"),
		Rating:     new(4.5),
		Notes:      new("Great"),
		CreatedAt:  time.Date(2026, 3, 1, 0, 0, 0, 0, time.UTC),
		UpdatedAt:  time.Date(2026, 3, 10, 0, 0, 0, 0, time.UTC),
	}
}

// ── Reads Add ─────────────────────────────────────────────────────────────────

func TestReadsAddPostsToBookEndpoint(t *testing.T) {
	const bookID = int64(1)
	var captured reads.CreateReadRequest
	var capturedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if answerLookup(w, r, "/api/books", bookID, books.Book{ID: bookID, Title: "Dune"}) {
			return
		}
		capturedPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Error(err)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(reads.Read{ID: 10, BookID: bookID})
	}))

	defer server.Close()

	exitCode, stdout, stderr := runCLI([]string{
		"reads", "add", "--book", "1", "--started-at", "2026-01-01",
		"--abandoned-at", "2026-01-03", "--rating", "4.5", "--notes", "Excellent",
		"--server", server.URL,
	})

	if exitCode != 0 || stderr != "" {
		t.Fatalf("unexpected result: exit=%d stderr=%q", exitCode, stderr)
	}
	if capturedPath != "/api/books/1/reads" {
		t.Fatalf("unexpected path %q", capturedPath)
	}
	if captured.StartedAt == nil || *captured.StartedAt != "2026-01-01" || captured.AbandonedAt == nil || *captured.AbandonedAt != "2026-01-03" || captured.Rating == nil || *captured.Rating != 4.5 {
		t.Fatalf("unexpected request: %#v", captured)
	}
	if stdout != "10\t1\n" {
		t.Fatalf("unexpected stdout %q", stdout)
	}
}

func TestReadsAddRequiresBook(t *testing.T) {
	exitCode, stdout, stderr := runCLI([]string{"reads", "add"})

	if exitCode != 2 || stdout != "" || !strings.Contains(stderr, "--book is required") {
		t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
}

// ── Reads Remove ──────────────────────────────────────────────────────────────

func TestReadsRemoveDeletesByIDAndPrintsResult(t *testing.T) {
	var method, path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	exitCode, stdout, stderr := runCLI([]string{"reads", "rm", "--server", server.URL, "12"})
	if exitCode != 0 || stderr != "" || stdout != "removed read 12\n" {
		t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if method != http.MethodDelete || path != "/api/reads/12" {
		t.Fatalf("expected DELETE /api/reads/12, got %s %s", method, path)
	}
}

func TestReadsRemoveRejectsInvalidIDsWithoutRequest(t *testing.T) {
	for _, ref := range []string{"Dune", "0", "-1"} {
		t.Run(ref, func(t *testing.T) {
			args := []string{"reads", "rm", ref}

			if strings.HasPrefix(ref, "-") {
				args = []string{"reads", "rm", "--", ref}
			}

			exitCode, stdout, stderr := runCLI(args)

			if exitCode != 2 || stdout != "" || !strings.Contains(stderr, "invalid") {
				t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
			}
		})
	}
}

// ── Reads Edit ───────────────────────────────────────────────────────────────

func TestReadsEditPatchesNullableFields(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/api/reads/9" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		_ = json.NewEncoder(w).Encode(reads.Read{ID: 9, BookID: 3})
	}))
	defer server.Close()

	code, stdout, stderr := runCLI([]string{"reads", "edit", "--rating", "4.5", "--clear", "notes", "--server", server.URL, "9"})
	if code != 0 || stdout != "9\t3\n" || stderr != "" || body["rating"] != 4.5 || body["notes"] != nil || len(body) != 2 {
		t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q body=%#v", code, stdout, stderr, body)
	}
}
