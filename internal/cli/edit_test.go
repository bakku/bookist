package cli_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strings"
	"testing"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/lists"
	"bakku.dev/bookist/internal/reads"
)

func TestEditAcceptsTargetBeforeOrAfterOptions(t *testing.T) {
	var received []map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch {
			t.Fatalf("expected PATCH, got %s", r.Method)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		received = append(received, payload)
		switch r.URL.Path {
		case "/api/books/12":
			_ = json.NewEncoder(w).Encode(books.Book{ID: 12, Title: "New"})
		case "/api/authors/6":
			_ = json.NewEncoder(w).Encode(authors.Author{ID: 6, Name: "New"})
		case "/api/lists/5":
			_ = json.NewEncoder(w).Encode(lists.List{ID: 5, Name: "New"})
		case "/api/reads/9":
			_ = json.NewEncoder(w).Encode(reads.Read{ID: 9, BookID: 3})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	for _, test := range []struct {
		name          string
		targetFirst   []string
		optionsFirst  []string
		expectedPatch map[string]any
	}{
		{name: "books", targetFirst: []string{"books", "edit", "12", "--title", "New"}, optionsFirst: []string{"books", "edit", "--title", "New", "12"}, expectedPatch: map[string]any{"title": "New"}},
		{name: "authors", targetFirst: []string{"authors", "edit", "6", "--name", "New"}, optionsFirst: []string{"authors", "edit", "--name", "New", "6"}, expectedPatch: map[string]any{"name": "New"}},
		{name: "lists", targetFirst: []string{"lists", "edit", "5", "--name", "New"}, optionsFirst: []string{"lists", "edit", "--name", "New", "5"}, expectedPatch: map[string]any{"name": "New"}},
		{name: "reads", targetFirst: []string{"reads", "edit", "9", "--rating", "4.5"}, optionsFirst: []string{"reads", "edit", "--rating", "4.5", "9"}, expectedPatch: map[string]any{"rating": 4.5}},
	} {
		t.Run(test.name, func(t *testing.T) {
			start := len(received)
			for _, args := range [][]string{test.targetFirst, test.optionsFirst} {
				args = append(args, "--server", server.URL)
				code, _, stderr := runCLI(args)
				if code != 0 || stderr != "" {
					t.Fatalf("%v: unexpected result: exit=%d stderr=%q", args, code, stderr)
				}
			}
			payloads := received[start:]
			if len(payloads) != 2 || !reflect.DeepEqual(payloads[0], payloads[1]) || !reflect.DeepEqual(payloads[0], test.expectedPatch) {
				t.Fatalf("expected equivalent payloads %#v, got %#v", test.expectedPatch, payloads)
			}
		})
	}
}

func TestEditParsingSupportsMixedEqualsRepeatableAndDelimiter(t *testing.T) {
	var bookPatch map[string]any
	var readPatch map[string]any
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/books/12":
			_ = json.NewDecoder(r.Body).Decode(&bookPatch)
			_ = json.NewEncoder(w).Encode(books.Book{ID: 12, Title: "New"})
		case "/api/reads/9":
			_ = json.NewDecoder(r.Body).Decode(&readPatch)
			_ = json.NewEncoder(w).Encode(reads.Read{ID: 9, BookID: 3})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	code, _, stderr := runCLI([]string{
		"books", "edit", "--server", server.URL, "12", "--title=New",
		"--author=4", "--author", "8", "--clear", "isbn",
	})
	if code != 0 || stderr != "" {
		t.Fatalf("mixed books edit failed: exit=%d stderr=%q", code, stderr)
	}
	expectedBookPatch := map[string]any{"title": "New", "author_ids": []any{float64(4), float64(8)}, "isbn": nil}
	if !reflect.DeepEqual(bookPatch, expectedBookPatch) {
		t.Fatalf("expected patch %#v, got %#v", expectedBookPatch, bookPatch)
	}

	code, _, stderr = runCLI([]string{"reads", "edit", "--server=" + server.URL, "--rating=4.5", "--", "9"})
	if code != 0 || stderr != "" {
		t.Fatalf("delimiter reads edit failed: exit=%d stderr=%q", code, stderr)
	}
	if !reflect.DeepEqual(readPatch, map[string]any{"rating": 4.5}) {
		t.Fatalf("unexpected read patch: %#v", readPatch)
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
		{"authors", "edit", "Le Guin", "--clear", "name", "--server", server.URL},
		{"books", "edit", "12", "--author", "4", "--clear", "authors", "--server", server.URL},
		{"books", "edit", "--title", "New", "--server", server.URL, "Dune", "extra"},
		{"books", "edit", "Dune", "--title"},
		{"books", "edit", "Dune", "--unknown", "value", "--server", server.URL},
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

func TestEditHelpShowsTargetFirstSyntaxAfterTarget(t *testing.T) {
	for _, test := range []struct {
		resource string
		target   string
		hasClear bool
	}{
		{resource: "books", target: "12", hasClear: true},
		{resource: "reads", target: "9", hasClear: true},
		{resource: "lists", target: "5", hasClear: true},
		{resource: "authors", target: "6", hasClear: false},
	} {
		code, stdout, stderr := runCLI([]string{test.resource, "edit", test.target, "--help"})
		if code != 0 || stderr != "" || !strings.Contains(stdout, "edit <") || !strings.Contains(stdout, "> [options]") {
			t.Fatalf("%s: unexpected help: exit=%d stdout=%q stderr=%q", test.resource, code, stdout, stderr)
		}
		if strings.Contains(stdout, "--clear string") != test.hasClear {
			t.Fatalf("%s: unexpected --clear help: %q", test.resource, stdout)
		}
	}
}

func TestEditHelpDocumentsPartialUpdatesAndClearFields(t *testing.T) {
	for _, test := range []struct {
		resource string
		expected []string
	}{
		{resource: "books", expected: []string{"Omitted fields are unchanged", "acquisition-source", "published-year", "--clear authors", "complete author list becomes Alice and Bob", "existing authors"}},
		{resource: "reads", expected: []string{"Omitted fields are unchanged", "abandoned-at, finished-at, notes, rating, started-at", "--clear rating --clear notes", "--clear abandoned-at --finished-at 2026-09-16"}},
		{resource: "lists", expected: []string{"Omitted fields are unchanged", "Clear a nullable field (repeatable): description", "--clear description"}},
	} {
		code, stdout, stderr := runCLI([]string{test.resource, "edit", "--help"})
		if code != 0 || stderr != "" {
			t.Fatalf("%s: unexpected help result: exit=%d stderr=%q", test.resource, code, stderr)
		}
		for _, expected := range test.expected {
			if !strings.Contains(stdout, expected) {
				t.Errorf("%s: expected help to contain %q, got:\n%s", test.resource, expected, stdout)
			}
		}
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
