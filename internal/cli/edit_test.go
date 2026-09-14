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

func TestBooksEditPatchesOnlyChanges(t *testing.T) {
	var method, path, contentType string
	var body map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method, path, contentType = r.Method, r.URL.Path, r.Header.Get("Content-Type")
		_ = json.NewDecoder(r.Body).Decode(&body)
		_ = json.NewEncoder(w).Encode(books.Book{ID: 12, Title: "Dune Messiah"})
	}))
	defer server.Close()

	code, stdout, stderr := runCLI([]string{"books", "edit", "--title", "Dune Messiah", "--pages", "256", "--author", "4", "--author", "8", "--server", server.URL, "12"})
	if code != 0 || stderr != "" || stdout != "12\tDune Messiah\n" {
		t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if method != http.MethodPatch || path != "/api/books/12" || contentType != "application/json" {
		t.Fatalf("unexpected request: %s %s content-type=%q", method, path, contentType)
	}
	expected := `{"author_ids":[4,8],"pages":256,"title":"Dune Messiah"}`
	encoded, _ := json.Marshal(body)
	if string(encoded) != expected {
		t.Fatalf("expected body %s, got %s", expected, encoded)
	}
}

func TestBooksEditClearSendsNullAndResolvesTitle(t *testing.T) {
	var patched map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/books":
			_ = json.NewEncoder(w).Encode([]books.Book{{ID: 7, Title: "Kindred"}})
		case r.Method == http.MethodPatch && r.URL.Path == "/api/books/7":
			_ = json.NewDecoder(r.Body).Decode(&patched)
			_ = json.NewEncoder(w).Encode(books.Book{ID: 7, Title: "Kindred"})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	code, _, stderr := runCLI([]string{"books", "edit", "--clear", "authors", "--clear", "cover", "--server", server.URL, "kindred"})
	if code != 0 || stderr != "" {
		t.Fatalf("unexpected result: exit=%d stderr=%q", code, stderr)
	}
	if len(patched) != 2 || patched["author_ids"] != nil || patched["cover"] != nil {
		t.Fatalf("expected only null author_ids and cover, got %#v", patched)
	}
}

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

func TestEditValidationHappensBeforeNetwork(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests++ }))
	defer server.Close()
	tests := [][]string{
		{"books", "edit", "--server", server.URL, "Dune"},
		{"books", "edit", "--isbn", "1", "--clear", "isbn", "--server", server.URL, "Dune"},
		{"books", "edit", "--author", "", "--server", server.URL, "Dune"},
		{"reads", "edit", "--clear", "unknown", "--server", server.URL, "1"},
		{"lists", "edit", "--clear", "name", "--server", server.URL, "Favorites"},
		{"authors", "edit", "--clear", "name", "--server", server.URL, "Le Guin"},
		{"books", "edit", "--title", "New", "--server", server.URL, "Dune", "extra"},
	}
	for _, args := range tests {
		code, stdout, stderr := runCLI(args)
		if code != 2 || stdout != "" || stderr == "" {
			t.Fatalf("%v: unexpected result: exit=%d stdout=%q stderr=%q", args, code, stdout, stderr)
		}
	}
	if requests != 0 {
		t.Fatalf("expected no requests, got %d", requests)
	}
}

func TestEditHelpShowsOptionFirstSyntax(t *testing.T) {
	for _, resource := range []string{"books", "reads", "lists", "authors"} {
		code, stdout, stderr := runCLI([]string{resource, "edit", "--help"})
		if code != 0 || stderr != "" || !strings.Contains(stdout, "edit [options] <") || !strings.Contains(stdout, "--clear string") {
			t.Fatalf("%s: unexpected help: exit=%d stdout=%q stderr=%q", resource, code, stdout, stderr)
		}
	}
}

func TestBooksEditDoesNotCreateMissingAuthors(t *testing.T) {
	var methods []string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		if r.Method == http.MethodGet && r.URL.Path == "/api/authors" {
			_ = json.NewEncoder(w).Encode([]authors.Author{})
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	code, stdout, stderr := runCLI([]string{"books", "edit", "--author", "Missing Author", "--server", server.URL, "12"})
	if code != 1 || stdout != "" || !strings.Contains(stderr, "author not found: Missing Author") {
		t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
	if len(methods) != 1 || methods[0] != http.MethodGet {
		t.Fatalf("expected only an author lookup, got methods %#v", methods)
	}
}

func TestEditReportsServerErrorBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "rating must use increments of 0.5", http.StatusBadRequest)
	}))
	defer server.Close()

	code, stdout, stderr := runCLI([]string{"reads", "edit", "--rating", "4.2", "--server", server.URL, "9"})
	if code != 1 || stdout != "" || !strings.Contains(stderr, "400 Bad Request: rating must use increments of 0.5") {
		t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", code, stdout, stderr)
	}
}
