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
	finishedAt := "2026-01-03"
	rating := 4.5
	notes := "Excellent"

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/books":
			_ = json.NewEncoder(w).Encode([]books.Book{{ID: "book-1", Title: "Dune"}})
		case "/api/books/book-1/reads":
			_ = json.NewEncoder(w).Encode([]reads.Read{{
				ID: "read-1", BookID: "book-1", StartedAt: &startedAt,
				FinishedAt: &finishedAt, Rating: &rating, Notes: &notes,
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
		{name: "default pretty", expected: "ID      STARTED_AT  FINISHED_AT  RATING  NOTES\nread-1  2026-01-01  2026-01-03   4.5     Excellent\n"},
		{name: "explicit TSV", format: "tsv", expected: "read-1\t2026-01-01\t2026-01-03\t4.5\tExcellent\n"},
		{name: "pretty", format: "pretty", expected: "ID      STARTED_AT  FINISHED_AT  RATING  NOTES\nread-1  2026-01-01  2026-01-03   4.5     Excellent\n"},
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
	const bookID = "550e8400-e29b-41d4-a716-446655440000"
	now := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]reads.Read{{ID: "read-1", BookID: bookID, CreatedAt: now, UpdatedAt: now}})
	}))

	defer server.Close()

	exitCode, stdout, stderr := runCLI([]string{"reads", "list", "--book", bookID, "--server", server.URL, "--format", "json"})

	if exitCode != 0 || stderr != "" {
		t.Fatalf("unexpected result: exit=%d stderr=%q", exitCode, stderr)
	}
	if !strings.Contains(stdout, `"started_at":null`) || !strings.Contains(stdout, `"book_id":"`+bookID+`"`) ||
		!strings.Contains(stdout, `"created_at":"2026-01-02T03:04:05Z"`) || !strings.Contains(stdout, `"updated_at":"2026-01-02T03:04:05Z"`) {
		t.Fatalf("unexpected JSON output: %s", stdout)
	}
}

// ── Reads Add ─────────────────────────────────────────────────────────────────

func TestReadsAddPostsToBookEndpoint(t *testing.T) {
	const bookID = "550e8400-e29b-41d4-a716-446655440000"
	var captured reads.CreateReadRequest
	var capturedPath string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		if err := json.NewDecoder(r.Body).Decode(&captured); err != nil {
			t.Error(err)
		}
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(reads.Read{ID: "read-1", BookID: bookID})
	}))

	defer server.Close()

	exitCode, stdout, stderr := runCLI([]string{
		"reads", "add", "--book", bookID, "--started-at", "2026-01-01",
		"--finished-at", "2026-01-03", "--rating", "4.5", "--notes", "Excellent",
		"--server", server.URL,
	})

	if exitCode != 0 || stderr != "" {
		t.Fatalf("unexpected result: exit=%d stderr=%q", exitCode, stderr)
	}
	if capturedPath != "/api/books/"+bookID+"/reads" {
		t.Fatalf("unexpected path %q", capturedPath)
	}
	if captured.StartedAt == nil || *captured.StartedAt != "2026-01-01" || captured.Rating == nil || *captured.Rating != 4.5 {
		t.Fatalf("unexpected request: %#v", captured)
	}
	if stdout != "read-1\t"+bookID+"\n" {
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
