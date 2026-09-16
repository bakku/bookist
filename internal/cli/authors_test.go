package cli_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/cli"
)

// ── Authors Add ────────────────────────────────────────────────────────────────

func TestAuthorsAddPrintsIDAndName(t *testing.T) {
	var capturedBody string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/authors" {
			var req authors.CreateAuthorRequest
			_ = json.NewDecoder(r.Body).Decode(&req)
			capturedBody = req.Name
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(authors.Author{ID: 10, Name: req.Name})
		}
	}))

	defer server.Close()

	var stdout, stderr strings.Builder
	exitCode := cli.Run([]string{"authors", "add", "--name", "Test Author", "--server", server.URL}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", exitCode, stderr.String())
	}
	if capturedBody != "Test Author" {
		t.Fatalf("expected POST body name 'Test Author', got %q", capturedBody)
	}
	if !strings.Contains(stdout.String(), "10\tTest Author") {
		t.Fatalf("expected stdout to contain '10\\tTest Author', got %q", stdout.String())
	}
}

// ── Authors List ───────────────────────────────────────────────────────────────

func TestAuthorsListTableFormats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/authors" {
			_ = json.NewEncoder(w).Encode([]authors.Author{
				{ID: 1, Name: "Author One"},
				{ID: 2, Name: "Author Two"},
			})
		}
	}))
	defer server.Close()

	tests := []struct {
		name     string
		format   string
		expected string
	}{
		{name: "default pretty", expected: "ID  NAME\n1   Author One\n2   Author Two\n"},
		{name: "explicit TSV", format: "tsv", expected: "1\tAuthor One\n2\tAuthor Two\n"},
		{name: "pretty", format: "pretty", expected: "ID  NAME\n1   Author One\n2   Author Two\n"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := []string{"authors", "ls", "--server", server.URL}
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

func TestAuthorsListJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]authors.Author{{ID: 1, Name: "Author One"}})
	}))
	defer server.Close()

	exitCode, stdout, stderr := runCLI([]string{"authors", "ls", "--server", server.URL, "--format", "json"})
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var listed []authors.Author
	if err := json.Unmarshal([]byte(stdout), &listed); err != nil {
		t.Fatalf("expected valid JSON, got %q: %v", stdout, err)
	}
	if len(listed) != 1 || listed[0].ID != 1 || listed[0].Name != "Author One" {
		t.Fatalf("expected complete author JSON, got %#v", listed)
	}
}

func TestAuthorsLSForwardsQuery(t *testing.T) {
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("q")
		_ = json.NewEncoder(w).Encode([]authors.Author{})
	}))
	defer server.Close()

	exitCode, _, stderr := runCLI([]string{"authors", "ls", "--query", "Octavia Butler", "--server", server.URL})
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", exitCode, stderr)
	}
	if gotQuery != "Octavia Butler" {
		t.Fatalf("expected query %q, got %q", "Octavia Butler", gotQuery)
	}
}

// ── Authors Remove ────────────────────────────────────────────────────────────

func TestAuthorsRemoveDeletesByIDAndPrintsResult(t *testing.T) {
	var method, path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	exitCode, stdout, stderr := runCLI([]string{"authors", "rm", "--server", server.URL, "12"})
	if exitCode != 0 || stderr != "" || stdout != "removed author 12\n" {
		t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if method != http.MethodDelete || path != "/api/authors/12" {
		t.Fatalf("expected DELETE /api/authors/12, got %s %s", method, path)
	}
}

func TestAuthorsRemoveResolvesExactName(t *testing.T) {
	var deletedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			if r.URL.Query().Get("q") != "frank HERBERT" {
				t.Fatalf("unexpected query %q", r.URL.Query().Get("q"))
			}
			_ = json.NewEncoder(w).Encode([]authors.Author{{ID: 2, Name: "Frank Herbert Jr."}, {ID: 3, Name: "Frank Herbert"}})
			return
		}
		deletedPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	exitCode, _, stderr := runCLI([]string{"authors", "rm", "--server", server.URL, "frank HERBERT"})
	if exitCode != 0 || stderr != "" || deletedPath != "/api/authors/3" {
		t.Fatalf("unexpected result: exit=%d path=%q stderr=%q", exitCode, deletedPath, stderr)
	}
}

func TestAuthorsRemoveDoesNotCreateMissingAuthor(t *testing.T) {
	postCalled := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			postCalled = true
		}
		_ = json.NewEncoder(w).Encode([]authors.Author{})
	}))
	defer server.Close()

	exitCode, stdout, stderr := runCLI([]string{"authors", "rm", "--server", server.URL, "Missing"})

	if exitCode == 0 || stdout != "" || !strings.Contains(stderr, "author not found: Missing") {
		t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}

	if postCalled {
		t.Fatalf("expected no author creation")
	}
}

func TestAuthorsRemoveRejectsAmbiguousName(t *testing.T) {
	deleteCalled := false

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodDelete {
			deleteCalled = true
		}
		_ = json.NewEncoder(w).Encode([]authors.Author{{ID: 1, Name: "Ada"}, {ID: 2, Name: "ada"}})
	}))

	defer server.Close()

	exitCode, _, stderr := runCLI([]string{"authors", "rm", "--server", server.URL, "Ada"})

	if exitCode == 0 || !strings.Contains(stderr, `author "Ada" exists multiple times; pass an author ID instead`) {
		t.Fatalf("unexpected result: exit=%d stderr=%q", exitCode, stderr)
	}

	if deleteCalled {
		t.Fatal("expected ambiguity to prevent DELETE")
	}
}

// ── Authors Edit ─────────────────────────────────────────────────────────────

func TestAuthorsEditResolvesNameAndPatches(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode([]authors.Author{{ID: 6, Name: "Ursula Le Guin"}})
			return
		}
		if r.Method != http.MethodPatch || r.URL.Path != "/api/authors/6" {
			t.Fatalf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		_ = json.NewDecoder(r.Body).Decode(&body)
		_ = json.NewEncoder(w).Encode(authors.Author{ID: 6, Name: "Ursula K. Le Guin"})
	}))
	defer server.Close()

	code, stdout, stderr := runCLI([]string{"authors", "edit", "--name", "Ursula K. Le Guin", "--server", server.URL, "Ursula Le Guin"})
	if code != 0 || stdout != "6\tUrsula K. Le Guin\n" || stderr != "" || len(body) != 1 || body["name"] != "Ursula K. Le Guin" {
		t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q body=%#v", code, stdout, stderr, body)
	}
}
