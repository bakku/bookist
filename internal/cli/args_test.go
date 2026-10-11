package cli_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/lists"
	"bakku.dev/bookist/internal/reads"
)

// ── Flags After Positionals ───────────────────────────────────────────────────

func TestFlagsAfterPositionalsAreParsed(t *testing.T) {
	var lastRequest string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		lastRequest = r.Method + " " + r.URL.Path

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/books" && r.URL.Query().Get("q") == "Dune":
			_ = json.NewEncoder(w).Encode([]books.Book{{ID: 7, Title: "Dune"}})
		case r.Method == http.MethodGet && r.URL.Path == "/api/books":
			_ = json.NewEncoder(w).Encode([]books.Book{})
		case r.Method == http.MethodGet && r.URL.Path == "/api/authors":
			_ = json.NewEncoder(w).Encode([]authors.Author{})
		case r.Method == http.MethodGet && r.URL.Path == "/api/lists":
			_ = json.NewEncoder(w).Encode([]lists.List{})
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/books/"):
			_ = json.NewEncoder(w).Encode(books.Book{ID: 7, Title: "Dune", Authors: []authors.Author{}})
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/authors/"):
			_ = json.NewEncoder(w).Encode(authors.Author{ID: 6, Name: "Ursula K. Le Guin"})
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/lists/"):
			_ = json.NewEncoder(w).Encode(lists.List{ID: 5, Name: "Best Books"})
		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/reads/"):
			_ = json.NewEncoder(w).Encode(reads.Read{ID: 9, BookID: 7})
		case r.Method == http.MethodDelete:
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	tests := []struct {
		name     string
		args     []string
		expected string
	}{
		{name: "books rm by ID", args: []string{"books", "rm", "7"}, expected: "DELETE /api/books/7"},
		{name: "books rm by title", args: []string{"books", "rm", "Dune"}, expected: "DELETE /api/books/7"},
		{name: "books show", args: []string{"books", "show", "Dune", "--format", "json"}, expected: "GET /api/books/7"},
		{name: "authors rm", args: []string{"authors", "rm", "6"}, expected: "DELETE /api/authors/6"},
		{name: "authors show", args: []string{"authors", "show", "6", "--format", "json"}, expected: "GET /api/authors/6"},
		{name: "lists rm", args: []string{"lists", "rm", "5"}, expected: "DELETE /api/lists/5"},
		{name: "lists show", args: []string{"lists", "show", "5", "--format", "json"}, expected: "GET /api/lists/5"},
		{name: "reads rm", args: []string{"reads", "rm", "9"}, expected: "DELETE /api/reads/9"},
		{name: "reads show", args: []string{"reads", "show", "9", "--format", "json"}, expected: "GET /api/reads/9"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			lastRequest = ""

			// The server flag comes after the positional reference; if it were ignored
			// the command would talk to the default localhost server instead.
			args := append(append([]string{}, test.args...), "--server", server.URL)
			exitCode, _, stderr := runCLI(args)

			if exitCode != 0 || stderr != "" {
				t.Fatalf("unexpected result: exit=%d stderr=%q", exitCode, stderr)
			}
			if lastRequest != test.expected {
				t.Fatalf("expected last request %q, got %q", test.expected, lastRequest)
			}
		})
	}
}

func TestFlagsAfterPositionalsStillReportMissingValues(t *testing.T) {
	exitCode, stdout, stderr := runCLI([]string{"books", "rm", "Dune", "--server"})

	if exitCode != 2 || stdout != "" || !strings.Contains(stderr, "flag needs an argument: -server") {
		t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
}

// ── Stray Positionals ─────────────────────────────────────────────────────────

func TestCommandsWithoutPositionalsRejectStrayArguments(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests++ }))
	defer server.Close()

	tests := []struct {
		command string
		args    []string
	}{
		{command: "serve", args: []string{"serve", "extra"}},
		{command: "migrate", args: []string{"migrate", "extra"}},
		{command: "books add", args: []string{"books", "add", "Dune", "--isbn", "123", "--server", server.URL}},
		{command: "books ls", args: []string{"books", "ls", "extra", "--server", server.URL}},
		{command: "authors add", args: []string{"authors", "add", "--name", "Ada", "extra", "--server", server.URL}},
		{command: "authors ls", args: []string{"authors", "ls", "extra", "--server", server.URL}},
		{command: "lists add", args: []string{"lists", "add", "extra", "--name", "Best Books", "--server", server.URL}},
		{command: "lists ls", args: []string{"lists", "ls", "extra", "--server", server.URL}},
		{command: "lists add-book", args: []string{"lists", "add-book", "--list", "Best Books", "extra", "--book", "Dune", "--server", server.URL}},
		{command: "lists rm-book", args: []string{"lists", "rm-book", "--list", "Best Books", "--book", "Dune", "extra", "--server", server.URL}},
		{command: "reads add", args: []string{"reads", "add", "--book", "Dune", "extra", "--server", server.URL}},
		{command: "reads ls", args: []string{"reads", "ls", "extra", "--book", "Dune", "--server", server.URL}},
	}
	for _, test := range tests {
		t.Run(test.command, func(t *testing.T) {
			exitCode, stdout, stderr := runCLI(test.args)

			if exitCode != 2 || stdout != "" {
				t.Fatalf("unexpected result: exit=%d stdout=%q", exitCode, stdout)
			}
			if !strings.Contains(stderr, "Error: "+test.command+" does not accept positional arguments") {
				t.Fatalf("expected positional argument error, got %q", stderr)
			}
			if !strings.Contains(stderr, "USAGE:") {
				t.Fatalf("expected usage in stderr, got %q", stderr)
			}
		})
	}

	if requests != 0 {
		t.Fatalf("expected no requests, got %d", requests)
	}
}
