package cli_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
