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
			json.NewDecoder(r.Body).Decode(&req)
			capturedBody = req.Name
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(authors.Author{ID: "new-uuid", Name: req.Name})
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
	if !strings.Contains(stdout.String(), "new-uuid\tTest Author") {
		t.Fatalf("expected stdout to contain 'new-uuid\\tTest Author', got %q", stdout.String())
	}
}

// ── Authors List ───────────────────────────────────────────────────────────────

func TestAuthorsListTableFormats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/authors" {
			_ = json.NewEncoder(w).Encode([]authors.Author{
				{ID: "id-1", Name: "Author One"},
				{ID: "id-2", Name: "Author Two"},
			})
		}
	}))
	defer server.Close()

	tests := []struct {
		name     string
		format   string
		expected string
	}{
		{name: "default pretty", expected: "ID    NAME\nid-1  Author One\nid-2  Author Two\n"},
		{name: "explicit TSV", format: "tsv", expected: "id-1\tAuthor One\nid-2\tAuthor Two\n"},
		{name: "pretty", format: "pretty", expected: "ID    NAME\nid-1  Author One\nid-2  Author Two\n"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := []string{"authors", "list", "--server", server.URL}
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
		_ = json.NewEncoder(w).Encode([]authors.Author{{ID: "id-1", Name: "Author One"}})
	}))
	defer server.Close()

	exitCode, stdout, stderr := runCLI([]string{"authors", "list", "--server", server.URL, "--format", "json"})
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
	if len(listed) != 1 || listed[0].ID != "id-1" || listed[0].Name != "Author One" {
		t.Fatalf("expected complete author JSON, got %#v", listed)
	}
}
