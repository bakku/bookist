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

func TestAuthorsListPrintsIDAndName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/authors" {
			json.NewEncoder(w).Encode([]authors.Author{
				{ID: "id-1", Name: "Author One"},
				{ID: "id-2", Name: "Author Two"},
			})
		}
	}))
	defer server.Close()

	var stdout, stderr strings.Builder
	exitCode := cli.Run([]string{"authors", "list", "--server", server.URL}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", exitCode, stderr.String())
	}
	if !strings.Contains(stdout.String(), "id-1\tAuthor One") {
		t.Fatalf("expected stdout to contain 'id-1\\tAuthor One', got %q", stdout.String())
	}
	if !strings.Contains(stdout.String(), "id-2\tAuthor Two") {
		t.Fatalf("expected stdout to contain 'id-2\\tAuthor Two', got %q", stdout.String())
	}
}
