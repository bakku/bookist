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
)

// ── Remove Help ───────────────────────────────────────────────────────────────

func TestRemoveCommandsAppearInHelp(t *testing.T) {
	tests := []struct {
		args     []string
		expected []string
	}{
		{args: []string{"books", "--help"}, expected: []string{"rm   Remove a book"}},
		{args: []string{"authors", "--help"}, expected: []string{"rm   Remove an author"}},
		{args: []string{"lists", "--help"}, expected: []string{"rm        Remove a book list", "rm-book   Remove a book from a list"}},
		{args: []string{"reads", "--help"}, expected: []string{"rm   Remove a read"}},
		{args: []string{"books", "rm", "--help"}, expected: []string{"bookist books rm - Remove a book", "<title-or-ID>", "--server string"}},
		{args: []string{"authors", "rm", "--help"}, expected: []string{"bookist authors rm - Remove an author", "<name-or-ID>"}},
		{args: []string{"lists", "rm", "--help"}, expected: []string{"bookist lists rm - Remove a book list", "<name-or-ID>"}},
		{args: []string{"reads", "rm", "--help"}, expected: []string{"bookist reads rm - Remove a read", "<read-ID>"}},
		{args: []string{"lists", "rm-book", "--help"}, expected: []string{"bookist lists rm-book - Remove a book from a list", "--list string", "--book string"}},
	}

	for _, test := range tests {
		t.Run(strings.Join(test.args, " "), func(t *testing.T) {
			exitCode, stdout, stderr := runCLI(test.args)
			if exitCode != 0 || stderr != "" {
				t.Fatalf("unexpected result: exit=%d stderr=%q", exitCode, stderr)
			}
			for _, expected := range test.expected {
				if !strings.Contains(stdout, expected) {
					t.Errorf("expected help to contain %q, got:\n%s", expected, stdout)
				}
			}
		})
	}
}

// ── Remove Requests ───────────────────────────────────────────────────────────

func TestEntityRemoveCommandsDeleteByIDAndPrintStableOutput(t *testing.T) {
	tests := []struct {
		resource string
		path     string
		output   string
	}{
		{resource: "books", path: "/api/books/12", output: "removed book 12\n"},
		{resource: "authors", path: "/api/authors/12", output: "removed author 12\n"},
		{resource: "lists", path: "/api/lists/12", output: "removed list 12\n"},
		{resource: "reads", path: "/api/reads/12", output: "removed read 12\n"},
	}

	for _, test := range tests {
		t.Run(test.resource, func(t *testing.T) {
			var method, path string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				method, path = r.Method, r.URL.Path
				w.WriteHeader(http.StatusNoContent)
			}))
			defer server.Close()

			exitCode, stdout, stderr := runCLI([]string{test.resource, "rm", "--server", server.URL, "12"})
			if exitCode != 0 || stderr != "" || stdout != test.output {
				t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
			}
			if method != http.MethodDelete || path != test.path {
				t.Fatalf("expected DELETE %s, got %s %s", test.path, method, path)
			}
		})
	}
}

func TestRemoveCommandsResolveExactTextReferences(t *testing.T) {
	tests := []struct {
		resource string
		ref      string
		lookup   string
		deleted  string
		encode   func(http.ResponseWriter)
	}{
		{resource: "books", ref: "dUnE", lookup: "/api/books", deleted: "/api/books/2", encode: func(w http.ResponseWriter) {
			_ = json.NewEncoder(w).Encode([]books.Book{{ID: 1, Title: "Dune Messiah"}, {ID: 2, Title: "Dune"}})
		}},
		{resource: "authors", ref: "frank HERBERT", lookup: "/api/authors", deleted: "/api/authors/3", encode: func(w http.ResponseWriter) {
			_ = json.NewEncoder(w).Encode([]authors.Author{{ID: 2, Name: "Frank Herbert Jr."}, {ID: 3, Name: "Frank Herbert"}})
		}},
		{resource: "lists", ref: "to READ", lookup: "/api/lists", deleted: "/api/lists/4", encode: func(w http.ResponseWriter) {
			_ = json.NewEncoder(w).Encode([]lists.List{{ID: 3, Name: "To Read Later"}, {ID: 4, Name: "To Read"}})
		}},
	}

	for _, test := range tests {
		t.Run(test.resource, func(t *testing.T) {
			var deleted string
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				switch r.Method {
				case http.MethodGet:
					if r.URL.Path != test.lookup || r.URL.Query().Get("q") != test.ref {
						t.Fatalf("unexpected lookup %s?q=%s", r.URL.Path, r.URL.Query().Get("q"))
					}
					test.encode(w)
				case http.MethodDelete:
					deleted = r.URL.Path
					w.WriteHeader(http.StatusNoContent)
				}
			}))
			defer server.Close()

			exitCode, _, stderr := runCLI([]string{test.resource, "rm", "--server", server.URL, test.ref})
			if exitCode != 0 || stderr != "" {
				t.Fatalf("unexpected result: exit=%d stderr=%q", exitCode, stderr)
			}
			if deleted != test.deleted {
				t.Fatalf("expected deletion path %q, got %q", test.deleted, deleted)
			}
		})
	}
}

func TestAuthorRemoveDoesNotCreateMissingAuthor(t *testing.T) {
	var postCount int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			postCount++
		}
		_ = json.NewEncoder(w).Encode([]authors.Author{})
	}))
	defer server.Close()

	exitCode, stdout, stderr := runCLI([]string{"authors", "rm", "--server", server.URL, "Missing"})
	if exitCode == 0 || stdout != "" || !strings.Contains(stderr, "author not found: Missing") {
		t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if postCount != 0 {
		t.Fatalf("expected no author creation, got %d POST requests", postCount)
	}
}

func TestRemoveAmbiguityRequiresID(t *testing.T) {
	tests := []struct {
		resource string
		ref      string
		response any
		error    string
	}{
		{resource: "books", ref: "Dune", response: []books.Book{{ID: 1, Title: "Dune"}, {ID: 2, Title: "dune"}}, error: `book "Dune" exists multiple times; pass a book ID instead`},
		{resource: "authors", ref: "Ada", response: []authors.Author{{ID: 1, Name: "Ada"}, {ID: 2, Name: "ada"}}, error: `author "Ada" exists multiple times; pass an author ID instead`},
		{resource: "lists", ref: "Owned", response: []lists.List{{ID: 1, Name: "Owned"}, {ID: 2, Name: "owned"}}, error: `list "Owned" exists multiple times; pass a list ID instead`},
	}

	for _, test := range tests {
		t.Run(test.resource, func(t *testing.T) {
			deleteCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.Method == http.MethodDelete {
					deleteCount++
				}
				_ = json.NewEncoder(w).Encode(test.response)
			}))
			defer server.Close()

			exitCode, _, stderr := runCLI([]string{test.resource, "rm", "--server", server.URL, test.ref})
			if exitCode == 0 || !strings.Contains(stderr, test.error) {
				t.Fatalf("unexpected result: exit=%d stderr=%q", exitCode, stderr)
			}
			if deleteCount != 0 {
				t.Fatal("expected ambiguity to prevent DELETE")
			}
		})
	}
}

func TestListsRemoveBookResolvesAndDeletesRelationship(t *testing.T) {
	var method, deletedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/lists":
			_ = json.NewEncoder(w).Encode([]lists.List{{ID: 4, Name: "Owned"}})
		case "/api/books":
			_ = json.NewEncoder(w).Encode([]books.Book{{ID: 7, Title: "Dune"}})
		default:
			method, deletedPath = r.Method, r.URL.Path
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()

	exitCode, stdout, stderr := runCLI([]string{"lists", "rm-book", "--server", server.URL, "--list", "Owned", "--book", "Dune"})
	if exitCode != 0 || stderr != "" || stdout != "removed book 7 from list 4\n" {
		t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if method != http.MethodDelete || deletedPath != "/api/lists/4/books/7" {
		t.Fatalf("expected DELETE /api/lists/4/books/7, got %s %s", method, deletedPath)
	}
}

// ── Remove Validation ─────────────────────────────────────────────────────────

func TestEntityRemoveCommandsRequireExactlyOneOperand(t *testing.T) {
	for _, resource := range []string{"books", "authors", "lists", "reads"} {
		for _, operands := range [][]string{nil, {"1", "2"}} {
			args := append([]string{resource, "rm"}, operands...)
			exitCode, stdout, stderr := runCLI(args)
			if exitCode != 2 || stdout != "" || !strings.Contains(stderr, "requires exactly one") {
				t.Fatalf("%s: unexpected result: exit=%d stdout=%q stderr=%q", resource, exitCode, stdout, stderr)
			}
		}
	}
}

func TestRemoveFlagsMustPrecedePositionalReference(t *testing.T) {
	exitCode, _, stderr := runCLI([]string{"books", "rm", "12", "--server", "http://example.test"})
	if exitCode != 2 || !strings.Contains(stderr, "requires exactly one") {
		t.Fatalf("unexpected result: exit=%d stderr=%q", exitCode, stderr)
	}
}

func TestReadsRemoveRejectsNonPositiveAndTextIDsWithoutRequest(t *testing.T) {
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

func TestListsRemoveBookRequiresBothFlags(t *testing.T) {
	for _, args := range [][]string{{"lists", "rm-book"}, {"lists", "rm-book", "--list", "1"}, {"lists", "rm-book", "--book", "2"}} {
		exitCode, stdout, stderr := runCLI(args)
		if exitCode != 2 || stdout != "" || !strings.Contains(stderr, "required") {
			t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
		}
	}
}

func TestRemoveCommandsRequireNoContentStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	exitCode, stdout, stderr := runCLI([]string{"books", "rm", "--server", server.URL, "1"})
	if exitCode != 1 || stdout != "" || !strings.Contains(stderr, "remove book: server returned 200 OK") {
		t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
}
