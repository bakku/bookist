package cli_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bakku.dev/bookist/internal/cli"
	"bakku.dev/bookist/internal/lists"
)

// ── Lists List ────────────────────────────────────────────────────────────────

func TestListsListTableFormats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/lists" {
			_ = json.NewEncoder(w).Encode([]lists.List{
				{ID: "id-1", Name: "Want to Buy"},
				{ID: "id-2", Name: "Nightstand"},
			})
		}
	}))
	defer server.Close()

	tests := []struct {
		name     string
		format   string
		expected string
	}{
		{name: "default TSV", expected: "id-1\tWant to Buy\nid-2\tNightstand\n"},
		{name: "explicit TSV", format: "tsv", expected: "id-1\tWant to Buy\nid-2\tNightstand\n"},
		{name: "pretty", format: "pretty", expected: "ID    NAME\nid-1  Want to Buy\nid-2  Nightstand\n"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := []string{"lists", "list", "--server", server.URL}
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

func TestListsListJSONPreservesNullableDescription(t *testing.T) {
	description := "Books to purchase"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]lists.List{
			{ID: "id-1", Name: "Want to Buy", Description: &description},
			{ID: "id-2", Name: "Nightstand", Description: nil},
		})
	}))
	defer server.Close()

	exitCode, stdout, stderr := runCLI([]string{"lists", "list", "--server", server.URL, "--format", "json"})
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", exitCode, stderr)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}

	var listed []lists.List
	if err := json.Unmarshal([]byte(stdout), &listed); err != nil {
		t.Fatalf("expected valid JSON, got %q: %v", stdout, err)
	}
	if len(listed) != 2 {
		t.Fatalf("expected 2 lists, got %d", len(listed))
	}
	if listed[0].Description == nil || *listed[0].Description != description {
		t.Fatalf("expected populated description, got %#v", listed[0].Description)
	}
	if listed[1].Description != nil {
		t.Fatalf("expected null description, got %#v", listed[1].Description)
	}
}

// ── Lists Add ─────────────────────────────────────────────────────────────────

func TestListsAddPrintsIDAndName(t *testing.T) {
	var capturedBody lists.CreateListRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/lists" {
			json.NewDecoder(r.Body).Decode(&capturedBody)
			w.WriteHeader(http.StatusCreated)
			json.NewEncoder(w).Encode(lists.List{ID: "new-uuid", Name: capturedBody.Name})
		}
	}))
	defer server.Close()

	var stdout, stderr strings.Builder
	exitCode := cli.Run([]string{"lists", "add", "--name", "Want to Buy", "--server", server.URL}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", exitCode, stderr.String())
	}
	if capturedBody.Name != "Want to Buy" {
		t.Fatalf("expected POST body name 'Want to Buy', got %q", capturedBody.Name)
	}
	if !strings.Contains(stdout.String(), "new-uuid\tWant to Buy") {
		t.Fatalf("expected stdout to contain 'new-uuid\\tWant to Buy', got %q", stdout.String())
	}
}

// ── Lists Add-Book ────────────────────────────────────────────────────────────

func TestListsAddBookResolvesListByName(t *testing.T) {
	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/lists":
			json.NewEncoder(w).Encode([]lists.List{
				{ID: "list-1", Name: "Want to Buy"},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/books":
			json.NewEncoder(w).Encode([]interface{}{})
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/lists/"):
			capturedPath = r.URL.Path
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()

	var stdout, stderr strings.Builder
	exitCode := cli.Run([]string{"lists", "add-book", "--list", "Want to Buy", "--book", "550e8400-e29b-41d4-a716-446655440000", "--server", server.URL}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", exitCode, stderr.String())
	}
	if capturedPath != "/api/lists/list-1/books" {
		t.Fatalf("expected POST to /api/lists/list-1/books, got %s", capturedPath)
	}
}

func TestListsAddBookResolvesListByUUID(t *testing.T) {
	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/lists/"):
			capturedPath = r.URL.Path
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()

	var stdout, stderr strings.Builder
	exitCode := cli.Run([]string{"lists", "add-book", "--list", "550e8400-e29b-41d4-a716-446655440000", "--book", "550e8400-e29b-41d4-a716-446655440001", "--server", server.URL}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", exitCode, stderr.String())
	}
	if capturedPath != "/api/lists/550e8400-e29b-41d4-a716-446655440000/books" {
		t.Fatalf("expected POST to /api/lists/550e8400-e29b-41d4-a716-446655440000/books, got %s", capturedPath)
	}
}

func TestListsAddBookResolvesBookByTitle(t *testing.T) {
	var capturedBody lists.AddBookToListRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/lists":
			json.NewEncoder(w).Encode([]lists.List{
				{ID: "list-1", Name: "Want to Buy"},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/books":
			json.NewEncoder(w).Encode([]struct {
				ID    string `json:"id"`
				Title string `json:"title"`
			}{
				{ID: "book-1", Title: "Dune"},
			})
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/lists/"):
			json.NewDecoder(r.Body).Decode(&capturedBody)
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()

	var stdout, stderr strings.Builder
	exitCode := cli.Run([]string{"lists", "add-book", "--list", "Want to Buy", "--book", "Dune", "--server", server.URL}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", exitCode, stderr.String())
	}
	if capturedBody.BookID != "book-1" {
		t.Fatalf("expected book_id 'book-1', got %q", capturedBody.BookID)
	}
}

func TestListsAddBookListNotFoundExitsNonZero(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/lists" {
			json.NewEncoder(w).Encode([]lists.List{})
		}
	}))
	defer server.Close()

	var stdout, stderr strings.Builder
	exitCode := cli.Run([]string{"lists", "add-book", "--list", "Nonexistent", "--book", "550e8400-e29b-41d4-a716-446655440000", "--server", server.URL}, &stdout, &stderr)

	if exitCode == 0 {
		t.Fatalf("expected non-zero exit code, got 0")
	}
	if !strings.Contains(stderr.String(), "list not found") {
		t.Fatalf("expected stderr to contain 'list not found', got %q", stderr.String())
	}
}

func TestListsAddBookBookNotFoundExitsNonZero(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/lists":
			json.NewEncoder(w).Encode([]lists.List{
				{ID: "list-1", Name: "Want to Buy"},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/books":
			json.NewEncoder(w).Encode([]struct {
				ID    string `json:"id"`
				Title string `json:"title"`
			}{})
		}
	}))
	defer server.Close()

	var stdout, stderr strings.Builder
	exitCode := cli.Run([]string{"lists", "add-book", "--list", "Want to Buy", "--book", "Nonexistent", "--server", server.URL}, &stdout, &stderr)

	if exitCode == 0 {
		t.Fatalf("expected non-zero exit code, got 0")
	}
	if !strings.Contains(stderr.String(), "book not found") {
		t.Fatalf("expected stderr to contain 'book not found', got %q", stderr.String())
	}
}
