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
			args := []string{"reads", "list", "--book", "Dune", "--server", server.URL}
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
		_ = json.NewEncoder(w).Encode([]reads.Read{{ID: 10, BookID: bookID, CreatedAt: now, UpdatedAt: now}})
	}))

	defer server.Close()

	exitCode, stdout, stderr := runCLI([]string{"reads", "list", "--book", "1", "--server", server.URL, "--format", "json"})

	if exitCode != 0 || stderr != "" {
		t.Fatalf("unexpected result: exit=%d stderr=%q", exitCode, stderr)
	}
	if !strings.Contains(stdout, `"started_at":null`) || !strings.Contains(stdout, `"abandoned_at":null`) || !strings.Contains(stdout, `"book_id":1`) ||
		!strings.Contains(stdout, `"created_at":"2026-01-02T03:04:05Z"`) || !strings.Contains(stdout, `"updated_at":"2026-01-02T03:04:05Z"`) {
		t.Fatalf("unexpected JSON output: %s", stdout)
	}
}

// ── Reads Add ─────────────────────────────────────────────────────────────────

func TestReadsAddPostsToBookEndpoint(t *testing.T) {
	const bookID = int64(1)
	var captured reads.CreateReadRequest
	var capturedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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

// ── Reads Book Requirement ────────────────────────────────────────────────────

func TestReadsCommandsRequireBook(t *testing.T) {
	for _, command := range []string{"list", "add"} {
		t.Run(command, func(t *testing.T) {
			exitCode, stdout, stderr := runCLI([]string{"reads", command})
			if exitCode != 2 || stdout != "" || !strings.Contains(stderr, "--book is required") {
				t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
			}
		})
	}
}
