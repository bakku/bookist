package cli_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// recordServer is a fake API server for one resource collection. It serves
// GET /api/<collection>/<id>, answers GET /api/<collection>?q= with the records
// whose name contains the query, and records DELETE requests and name searches.
type recordServer struct {
	*httptest.Server

	deleted  string
	searched bool
}

func newRecordServer(t *testing.T, collection, nameField string, records map[int64]string) *recordServer {
	t.Helper()

	fake := &recordServer{}
	fake.Server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		record := func(id int64) map[string]any {
			return map[string]any{"id": id, nameField: records[id]}
		}

		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/"+collection:
			fake.searched = true
			found := []map[string]any{}
			for id, name := range records {
				if strings.Contains(strings.ToLower(name), strings.ToLower(r.URL.Query().Get("q"))) {
					found = append(found, record(id))
				}
			}
			sort.Slice(found, func(i, j int) bool { return found[i]["id"].(int64) < found[j]["id"].(int64) })
			_ = json.NewEncoder(w).Encode(found)

		case r.Method == http.MethodGet && strings.HasPrefix(r.URL.Path, "/api/"+collection+"/"):
			id, _ := strconv.ParseInt(strings.TrimPrefix(r.URL.Path, "/api/"+collection+"/"), 10, 64)
			if _, ok := records[id]; !ok {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusNotFound)
				_, _ = w.Write([]byte(`{"error":{"code":"not_found","message":"not found"}}`))
				return
			}
			_ = json.NewEncoder(w).Encode(record(id))

		case r.Method == http.MethodDelete:
			fake.deleted = r.URL.Path
			w.WriteHeader(http.StatusNoContent)

		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(fake.Close)

	return fake
}

// ── Numeric Book References ───────────────────────────────────────────────────

func TestBooksRemoveResolvesNumericReference(t *testing.T) {
	tests := []struct {
		name     string
		records  map[int64]string
		ref      string
		expected string
	}{
		{name: "title match only", records: map[int64]string{7: "1984", 8: "1984 Revisited"}, ref: "1984", expected: "/api/books/7"},
		{name: "ID match only", records: map[int64]string{1984: "Dune", 7: "Dune Messiah"}, ref: "1984", expected: "/api/books/1984"},
		{name: "ID and title match the same book", records: map[int64]string{1984: "1984", 7: "Dune"}, ref: "1984", expected: "/api/books/1984"},
		{name: "explicit ID ignores a matching title", records: map[int64]string{1984: "Dune", 7: "1984"}, ref: "id:1984", expected: "/api/books/1984"},
		{name: "explicit ID", records: map[int64]string{12: "Dune"}, ref: "id:12", expected: "/api/books/12"},
		{name: "zero is a title", records: map[int64]string{3: "0"}, ref: "0", expected: "/api/books/3"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := newRecordServer(t, "books", "title", test.records)

			exitCode, stdout, stderr := runCLI([]string{"books", "rm", test.ref, "--server", server.URL})

			if exitCode != 0 || stderr != "" || !strings.HasPrefix(stdout, "removed book ") {
				t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
			}
			if server.deleted != test.expected {
				t.Fatalf("expected DELETE %s, got %q", test.expected, server.deleted)
			}
		})
	}
}

func TestBooksRemoveRejectsAmbiguousNumericReference(t *testing.T) {
	server := newRecordServer(t, "books", "title", map[int64]string{1984: "Dune", 7: "1984"})

	exitCode, stdout, stderr := runCLI([]string{"books", "rm", "1984", "--server", server.URL})

	expected := `ambiguous book reference "1984": matches book 1984 ("Dune") and book 7 ("1984"); use id:1984 or id:7`
	if exitCode != 1 || stdout != "" || strings.TrimSpace(stderr) != expected {
		t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if server.deleted != "" {
		t.Fatalf("expected ambiguity to prevent DELETE, got %q", server.deleted)
	}
}

func TestBooksRemoveListsEveryAmbiguousCandidate(t *testing.T) {
	server := newRecordServer(t, "books", "title", map[int64]string{5: "Dune", 6: "5", 8: "5"})

	exitCode, _, stderr := runCLI([]string{"books", "rm", "5", "--server", server.URL})

	expected := `ambiguous book reference "5": matches book 5 ("Dune"), book 6 ("5") and book 8 ("5"); use id:5, id:6 or id:8`
	if exitCode != 1 || strings.TrimSpace(stderr) != expected {
		t.Fatalf("unexpected result: exit=%d stderr=%q", exitCode, stderr)
	}
	if server.deleted != "" {
		t.Fatalf("expected ambiguity to prevent DELETE, got %q", server.deleted)
	}
}

func TestBooksRemoveRejectsMultipleExactTitlesForNumericReference(t *testing.T) {
	server := newRecordServer(t, "books", "title", map[int64]string{7: "1984", 9: "1984"})

	exitCode, _, stderr := runCLI([]string{"books", "rm", "1984", "--server", server.URL})

	if exitCode != 1 || !strings.Contains(stderr, `book "1984" exists multiple times; pass a book ID instead`) {
		t.Fatalf("unexpected result: exit=%d stderr=%q", exitCode, stderr)
	}
	if server.deleted != "" {
		t.Fatalf("expected ambiguity to prevent DELETE, got %q", server.deleted)
	}
}

func TestBooksRemoveExplicitIDMustExist(t *testing.T) {
	server := newRecordServer(t, "books", "title", map[int64]string{7: "1984"})

	exitCode, stdout, stderr := runCLI([]string{"books", "rm", "id:1984", "--server", server.URL})

	if exitCode != 1 || stdout != "" || strings.TrimSpace(stderr) != "book not found: id:1984" {
		t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if server.deleted != "" || server.searched {
		t.Fatalf("expected no delete or title search, got deleted=%q searched=%t", server.deleted, server.searched)
	}
}

func TestBooksRemoveRejectsInvalidExplicitID(t *testing.T) {
	for _, ref := range []string{"id:", "id:abc", "id:0", "id:-3", "id:999999999999999999999999"} {
		t.Run(ref, func(t *testing.T) {
			server := newRecordServer(t, "books", "title", map[int64]string{})

			exitCode, stdout, stderr := runCLI([]string{"books", "rm", ref, "--server", server.URL})

			if exitCode != 1 || stdout != "" || strings.TrimSpace(stderr) != `invalid ID "`+ref+`"` {
				t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
			}
		})
	}
}

func TestBooksRemoveUnknownNumericReferenceIsNotFound(t *testing.T) {
	server := newRecordServer(t, "books", "title", map[int64]string{7: "Dune"})

	exitCode, _, stderr := runCLI([]string{"books", "rm", "1984", "--server", server.URL})

	if exitCode != 1 || strings.TrimSpace(stderr) != "book not found: 1984" {
		t.Fatalf("unexpected result: exit=%d stderr=%q", exitCode, stderr)
	}
}

// ── Numeric Author And List References ────────────────────────────────────────

func TestNumericReferencesAreAmbiguousForAuthorsAndLists(t *testing.T) {
	tests := []struct {
		name       string
		collection string
		nameField  string
		args       []string
		expected   string
	}{
		{
			name: "authors rm", collection: "authors", nameField: "name", args: []string{"authors", "rm", "2024"},
			expected: `ambiguous author reference "2024": matches author 2024 ("Ada") and author 3 ("2024"); use id:2024 or id:3`,
		},
		{
			name: "lists rm", collection: "lists", nameField: "name", args: []string{"lists", "rm", "2024"},
			expected: `ambiguous list reference "2024": matches list 2024 ("Ada") and list 3 ("2024"); use id:2024 or id:3`,
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := newRecordServer(t, test.collection, test.nameField, map[int64]string{2024: "Ada", 3: "2024"})

			exitCode, stdout, stderr := runCLI(append(test.args, "--server", server.URL))

			if exitCode != 1 || stdout != "" || strings.TrimSpace(stderr) != test.expected {
				t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
			}
			if server.deleted != "" {
				t.Fatalf("expected ambiguity to prevent DELETE, got %q", server.deleted)
			}
		})
	}
}

func TestNumericReferencesResolveByTitleOrExplicitIDForAuthorsAndLists(t *testing.T) {
	tests := []struct {
		name       string
		collection string
		records    map[int64]string
		args       []string
		expected   string
	}{
		{name: "author name", collection: "authors", records: map[int64]string{3: "2024"}, args: []string{"authors", "rm", "2024"}, expected: "/api/authors/3"},
		{name: "author explicit ID", collection: "authors", records: map[int64]string{2024: "Ada", 3: "2024"}, args: []string{"authors", "rm", "id:2024"}, expected: "/api/authors/2024"},
		{name: "list name", collection: "lists", records: map[int64]string{3: "2024"}, args: []string{"lists", "rm", "2024"}, expected: "/api/lists/3"},
		{name: "list explicit ID", collection: "lists", records: map[int64]string{2024: "Ada", 3: "2024"}, args: []string{"lists", "rm", "id:2024"}, expected: "/api/lists/2024"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := newRecordServer(t, test.collection, "name", test.records)

			exitCode, _, stderr := runCLI(append(test.args, "--server", server.URL))

			if exitCode != 0 || stderr != "" {
				t.Fatalf("unexpected result: exit=%d stderr=%q", exitCode, stderr)
			}
			if server.deleted != test.expected {
				t.Fatalf("expected DELETE %s, got %q", test.expected, server.deleted)
			}
		})
	}
}

func TestListsAddBookResolvesNumericTitleAndExplicitIDs(t *testing.T) {
	var posted string
	var bookID int64
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodGet && r.URL.Path == "/api/books":
			_ = json.NewEncoder(w).Encode([]map[string]any{{"id": 7, "title": "1984"}})
		case r.Method == http.MethodGet && r.URL.Path == "/api/lists/4":
			_ = json.NewEncoder(w).Encode(map[string]any{"id": 4, "name": "Owned"})
		case r.Method == http.MethodPost:
			posted = r.URL.Path
			var body struct {
				BookID int64 `json:"book_id"`
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			bookID = body.BookID
			w.WriteHeader(http.StatusNoContent)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	exitCode, stdout, stderr := runCLI([]string{"lists", "add-book", "--list", "id:4", "--book", "1984", "--server", server.URL})

	if exitCode != 0 || stderr != "" || stdout != "added book 7 to list 4\n" {
		t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if posted != "/api/lists/4/books" || bookID != 7 {
		t.Fatalf("expected book 7 posted to /api/lists/4/books, got book %d at %q", bookID, posted)
	}
}
