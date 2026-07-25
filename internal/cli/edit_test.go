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

	code, stdout, stderr := runCLI([]string{"books", "edit", "12", "--title", "Dune Messiah", "--pages", "256", "--author", "4", "--author", "8", "--server", server.URL})
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

	code, _, stderr := runCLI([]string{"books", "edit", "kindred", "--clear", "authors", "--clear", "cover", "--server", server.URL})
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
	code, stdout, stderr := runCLI([]string{"reads", "edit", "9", "--rating", "4.5", "--clear", "notes", "--server", server.URL})
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
	code, stdout, stderr := runCLI([]string{"lists", "edit", "favorites", "--name", "Best", "--clear", "description", "--server", server.URL})
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
	code, stdout, stderr := runCLI([]string{"authors", "edit", "Ursula Le Guin", "--name", "Ursula K. Le Guin", "--server", server.URL})
	if code != 0 || stdout != "6\tUrsula K. Le Guin\n" || stderr != "" || len(body) != 1 || body["name"] != "Ursula K. Le Guin" {
		t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q body=%#v", code, stdout, stderr, body)
	}
}

func TestEditValidationHappensBeforeNetwork(t *testing.T) {
	requests := 0
	server := httptest.NewServer(http.HandlerFunc(func(http.ResponseWriter, *http.Request) { requests++ }))
	defer server.Close()
	tests := [][]string{
		{"books", "edit", "Dune", "--server", server.URL},
		{"books", "edit", "Dune", "--isbn", "1", "--clear", "isbn", "--server", server.URL},
		{"books", "edit", "Dune", "--author", "", "--server", server.URL},
		{"reads", "edit", "1", "--clear", "unknown", "--server", server.URL},
		{"lists", "edit", "Favorites", "--clear", "name", "--server", server.URL},
		{"authors", "edit", "Le Guin", "--clear", "name", "--server", server.URL},
		{"books", "edit", "Dune", "extra", "--title", "New", "--server", server.URL},
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

func TestEditHelpShowsTargetFirstSyntax(t *testing.T) {
	for _, resource := range []string{"books", "reads", "lists", "authors"} {
		code, stdout, stderr := runCLI([]string{resource, "edit", "--help"})
		if code != 0 || stderr != "" || !strings.Contains(stdout, "edit <") || !strings.Contains(stdout, "[options]") || !strings.Contains(stdout, "--clear string") {
			t.Fatalf("%s: unexpected help: exit=%d stdout=%q stderr=%q", resource, code, stdout, stderr)
		}
	}
}
