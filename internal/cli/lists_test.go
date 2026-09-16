package cli_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/cli"
	"bakku.dev/bookist/internal/lists"
)

// ── Lists List ────────────────────────────────────────────────────────────────

func TestListsListTableFormats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/lists" {
			_ = json.NewEncoder(w).Encode([]lists.List{
				{ID: 1, Name: "Want to Buy"},
				{ID: 2, Name: "Nightstand"},
			})
		}
	}))
	defer server.Close()

	tests := []struct {
		name     string
		format   string
		expected string
	}{
		{name: "default pretty", expected: "ID  NAME\n1   Want to Buy\n2   Nightstand\n"},
		{name: "explicit TSV", format: "tsv", expected: "1\tWant to Buy\n2\tNightstand\n"},
		{name: "pretty", format: "pretty", expected: "ID  NAME\n1   Want to Buy\n2   Nightstand\n"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			args := []string{"lists", "ls", "--server", server.URL}
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
			{ID: 1, Name: "Want to Buy", Description: &description},
			{ID: 2, Name: "Nightstand", Description: nil},
		})
	}))
	defer server.Close()

	exitCode, stdout, stderr := runCLI([]string{"lists", "ls", "--server", server.URL, "--format", "json"})
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

func TestListsLSForwardsQuery(t *testing.T) {
	var gotQuery string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotQuery = r.URL.Query().Get("q")
		_ = json.NewEncoder(w).Encode([]lists.List{})
	}))
	defer server.Close()

	exitCode, _, stderr := runCLI([]string{"lists", "ls", "--query", "Want & Buy", "--server", server.URL})
	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", exitCode, stderr)
	}
	if gotQuery != "Want & Buy" {
		t.Fatalf("expected query %q, got %q", "Want & Buy", gotQuery)
	}
}

// ── Lists Add ─────────────────────────────────────────────────────────────────

func TestListsAddPrintsIDAndName(t *testing.T) {
	var capturedBody lists.CreateListRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost && r.URL.Path == "/api/lists" {
			_ = json.NewDecoder(r.Body).Decode(&capturedBody)
			w.WriteHeader(http.StatusCreated)
			_ = json.NewEncoder(w).Encode(lists.List{ID: 10, Name: capturedBody.Name})
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
	if !strings.Contains(stdout.String(), "10\tWant to Buy") {
		t.Fatalf("expected stdout to contain '10\\tWant to Buy', got %q", stdout.String())
	}
}

// ── Lists Add-Book ────────────────────────────────────────────────────────────

func TestListsAddBookResolvesListByName(t *testing.T) {
	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/lists":
			if got := r.URL.Query().Get("q"); got != "want TO buy" {
				t.Fatalf("expected list query %q, got %q", "want TO buy", got)
			}
			_ = json.NewEncoder(w).Encode([]lists.List{
				{ID: 9, Name: "Want to Buy Later"},
				{ID: 1, Name: "Want to Buy"},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/books":
			_ = json.NewEncoder(w).Encode([]interface{}{})
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/lists/"):
			capturedPath = r.URL.Path
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()

	var stdout, stderr strings.Builder
	exitCode := cli.Run([]string{"lists", "add-book", "--list", "want TO buy", "--book", "2", "--server", server.URL}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", exitCode, stderr.String())
	}
	if capturedPath != "/api/lists/1/books" {
		t.Fatalf("expected POST to /api/lists/1/books, got %s", capturedPath)
	}
}

func TestListsAddBookPassesIntegerIDsThrough(t *testing.T) {
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
	exitCode := cli.Run([]string{"lists", "add-book", "--list", "10", "--book", "20", "--server", server.URL}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", exitCode, stderr.String())
	}
	if capturedPath != "/api/lists/10/books" {
		t.Fatalf("expected POST to /api/lists/10/books, got %s", capturedPath)
	}
}

func TestListsAddBookResolvesBookByTitle(t *testing.T) {
	var capturedBody lists.AddBookToListRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/lists":
			_ = json.NewEncoder(w).Encode([]lists.List{
				{ID: 1, Name: "Want to Buy"},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/books":
			if got := r.URL.Query().Get("q"); got != "dUnE" {
				t.Fatalf("expected book query %q, got %q", "dUnE", got)
			}
			_ = json.NewEncoder(w).Encode([]struct {
				ID    int64  `json:"id"`
				Title string `json:"title"`
			}{
				{ID: 3, Title: "Dune Messiah"},
				{ID: 2, Title: "Dune"},
			})
		case r.Method == http.MethodPost && strings.HasPrefix(r.URL.Path, "/api/lists/"):
			_ = json.NewDecoder(r.Body).Decode(&capturedBody)
			w.WriteHeader(http.StatusNoContent)
		}
	}))
	defer server.Close()

	var stdout, stderr strings.Builder
	exitCode := cli.Run([]string{"lists", "add-book", "--list", "Want to Buy", "--book", "dUnE", "--server", server.URL}, &stdout, &stderr)

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d; stderr: %s", exitCode, stderr.String())
	}
	if capturedBody.BookID != 2 {
		t.Fatalf("expected book_id 2, got %d", capturedBody.BookID)
	}
}

func TestListsAddBookWithAmbiguousTitleRequiresID(t *testing.T) {
	bookPosted := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/lists":
			_ = json.NewEncoder(w).Encode([]lists.List{{ID: 1, Name: "Want to Buy"}})
		case r.Method == http.MethodGet && r.URL.Path == "/api/books":
			_ = json.NewEncoder(w).Encode([]struct {
				ID    int64  `json:"id"`
				Title string `json:"title"`
			}{
				{ID: 1, Title: "Dune"},
				{ID: 2, Title: "dUnE"},
			})
		case r.Method == http.MethodPost:
			bookPosted = true
		}
	}))
	defer server.Close()

	var stdout, stderr strings.Builder
	exitCode := cli.Run([]string{"lists", "add-book", "--list", "Want to Buy", "--book", "Dune", "--server", server.URL}, &stdout, &stderr)

	if exitCode == 0 {
		t.Fatal("expected non-zero exit code")
	}
	if bookPosted {
		t.Fatal("expected ambiguous title to prevent adding a book")
	}
	if !strings.Contains(stderr.String(), `book "Dune" exists multiple times; pass a book ID instead`) {
		t.Fatalf("unexpected error: %q", stderr.String())
	}
}

func TestListsAddBookListNotFoundExitsNonZero(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && r.URL.Path == "/api/lists" {
			_ = json.NewEncoder(w).Encode([]lists.List{})
		}
	}))
	defer server.Close()

	var stdout, stderr strings.Builder
	exitCode := cli.Run([]string{"lists", "add-book", "--list", "Nonexistent", "--book", "1", "--server", server.URL}, &stdout, &stderr)

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
			_ = json.NewEncoder(w).Encode([]lists.List{
				{ID: 1, Name: "Want to Buy"},
			})
		case r.Method == http.MethodGet && r.URL.Path == "/api/books":
			_ = json.NewEncoder(w).Encode([]struct {
				ID    int64  `json:"id"`
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

// ── Lists Remove ──────────────────────────────────────────────────────────────

func TestListsRemoveDeletesByIDAndPrintsResult(t *testing.T) {
	var method, path string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path = r.Method, r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	exitCode, stdout, stderr := runCLI([]string{"lists", "rm", "--server", server.URL, "12"})
	if exitCode != 0 || stderr != "" || stdout != "removed list 12\n" {
		t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if method != http.MethodDelete || path != "/api/lists/12" {
		t.Fatalf("expected DELETE /api/lists/12, got %s %s", method, path)
	}
}

func TestListsRemoveResolvesExactName(t *testing.T) {
	var deletedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			_ = json.NewEncoder(w).Encode([]lists.List{{ID: 3, Name: "To Read Later"}, {ID: 4, Name: "To Read"}})
			return
		}
		deletedPath = r.URL.Path
		w.WriteHeader(http.StatusNoContent)
	}))
	defer server.Close()

	exitCode, _, stderr := runCLI([]string{"lists", "rm", "--server", server.URL, "to READ"})
	if exitCode != 0 || stderr != "" || deletedPath != "/api/lists/4" {
		t.Fatalf("unexpected result: exit=%d path=%q stderr=%q", exitCode, deletedPath, stderr)
	}
}

func TestListsRemoveRejectsAmbiguousName(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]lists.List{{ID: 1, Name: "Owned"}, {ID: 2, Name: "owned"}})
	}))
	defer server.Close()

	exitCode, _, stderr := runCLI([]string{"lists", "rm", "--server", server.URL, "Owned"})
	if exitCode == 0 || !strings.Contains(stderr, `list "Owned" exists multiple times; pass a list ID instead`) {
		t.Fatalf("unexpected result: exit=%d stderr=%q", exitCode, stderr)
	}
}

// ── Lists Remove-Book ─────────────────────────────────────────────────────────

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

func TestListsRemoveBookRequiresBothFlags(t *testing.T) {
	for _, args := range [][]string{{"lists", "rm-book"}, {"lists", "rm-book", "--list", "1"}, {"lists", "rm-book", "--book", "2"}} {
		exitCode, stdout, stderr := runCLI(args)
		if exitCode != 2 || stdout != "" || !strings.Contains(stderr, "required") {
			t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
		}
	}
}

// ── Lists Edit ───────────────────────────────────────────────────────────────

func TestListsEditResolvesNameAndPatches(t *testing.T) {
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case http.MethodGet:
			_ = json.NewEncoder(w).Encode([]lists.List{{ID: 5, Name: "Favorites"}})
		case http.MethodPatch:
			if r.URL.Path != "/api/lists/5" {
				t.Fatalf("unexpected path %s", r.URL.Path)
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			_ = json.NewEncoder(w).Encode(lists.List{ID: 5, Name: "Best"})
		}
	}))
	defer server.Close()

	code, stdout, stderr := runCLI([]string{"lists", "edit", "--name", "Best", "--clear", "description", "--server", server.URL, "favorites"})
	if code != 0 || stdout != "5\tBest\n" || stderr != "" || body["name"] != "Best" || body["description"] != nil || len(body) != 2 {
		t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q body=%#v", code, stdout, stderr, body)
	}
}
