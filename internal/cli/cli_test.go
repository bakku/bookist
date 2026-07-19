package cli_test

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bakku.dev/bookist/internal/cli"
)

// ── Help Display ──────────────────────────────────────────────────────────────

func TestRootHelpShowsOnlyTopLevelCommands(t *testing.T) {
	for _, args := range [][]string{{}, {"--help"}, {"-h"}, {"help"}, {"h"}} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			exitCode, stdout, stderr := runCLI(args)

			if exitCode != 0 {
				t.Fatalf("expected exit code 0, got %d", exitCode)
			}
			if stderr != "" {
				t.Fatalf("expected empty stderr, got %q", stderr)
			}

			for _, expected := range []string{
				"NAME:\n   bookist - Manage a home library",
				"serve    Start the Bookist server",
				"migrate  Run database migrations",
				"books    Manage books",
				"authors  Manage authors",
				"lists    Manage book lists",
				"reads    Manage book reads",
			} {
				if !strings.Contains(stdout, expected) {
					t.Errorf("expected stdout to contain %q, got:\n%s", expected, stdout)
				}
			}

			for _, unexpected := range []string{"bookist books list", "bookist authors add", "--server"} {
				if strings.Contains(stdout, unexpected) {
					t.Errorf("expected stdout not to contain %q, got:\n%s", unexpected, stdout)
				}
			}
		})
	}
}

func TestParentHelpShowsOnlyImmediateCommands(t *testing.T) {
	tests := []struct {
		name       string
		args       []string
		expected   []string
		unexpected []string
	}{
		{
			name:       "books",
			args:       []string{"books", "--help"},
			expected:   []string{"bookist books - Manage books", "list  List books", "add   Add a book"},
			unexpected: []string{"Manage authors", "add-book", "--title"},
		},
		{
			name:       "authors",
			args:       []string{"authors", "--help"},
			expected:   []string{"bookist authors - Manage authors", "list  List authors", "add   Add an author"},
			unexpected: []string{"Manage books", "add-book", "--name"},
		},
		{
			name:       "lists",
			args:       []string{"lists", "--help"},
			expected:   []string{"bookist lists - Manage book lists", "list      List book lists", "add       Add a book list", "add-book  Add a book to a list"},
			unexpected: []string{"Manage authors", "--description", "--book"},
		},
		{
			name:       "reads",
			args:       []string{"reads", "--help"},
			expected:   []string{"bookist reads - Manage book reads", "list  List reads for a book", "add   Record a read for a book"},
			unexpected: []string{"Manage authors", "--rating", "--book"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			exitCode, stdout, stderr := runCLI(test.args)

			if exitCode != 0 {
				t.Fatalf("expected exit code 0, got %d", exitCode)
			}
			if stderr != "" {
				t.Fatalf("expected empty stderr, got %q", stderr)
			}
			for _, expected := range test.expected {
				if !strings.Contains(stdout, expected) {
					t.Errorf("expected stdout to contain %q, got:\n%s", expected, stdout)
				}
			}
			for _, unexpected := range test.unexpected {
				if strings.Contains(stdout, unexpected) {
					t.Errorf("expected stdout not to contain %q, got:\n%s", unexpected, stdout)
				}
			}
		})
	}
}

func TestLeafHelpShowsCommandOptionsAndExitsSuccessfully(t *testing.T) {
	tests := []struct {
		name     string
		args     []string
		expected []string
	}{
		{name: "serve", args: []string{"serve", "--help"}, expected: []string{"bookist serve - Start the Bookist server", "--addr string", "--db string"}},
		{name: "migrate", args: []string{"migrate", "--help"}, expected: []string{"bookist migrate - Run database migrations", "--db string"}},
		{name: "books list", args: []string{"books", "list", "--help"}, expected: []string{"bookist books list - List books", "--format string", "Output format (tsv|pretty|json) (default: pretty)", "--server string"}},
		{name: "books add", args: []string{"books", "add", "-h"}, expected: []string{"bookist books add - Add a book", "--author string", "--title string"}},
		{name: "books add long single dash", args: []string{"books", "add", "-help"}, expected: []string{"bookist books add - Add a book", "--author string", "--title string"}},
		{name: "authors list", args: []string{"authors", "list", "--help"}, expected: []string{"bookist authors list - List authors", "--format string", "Output format (tsv|pretty|json) (default: pretty)", "--server string"}},
		{name: "authors add", args: []string{"authors", "add", "--help"}, expected: []string{"bookist authors add - Add an author", "--name string"}},
		{name: "lists list", args: []string{"lists", "list", "--help"}, expected: []string{"bookist lists list - List book lists", "--format string", "Output format (tsv|pretty|json) (default: pretty)", "--server string"}},
		{name: "lists add-book", args: []string{"lists", "add-book", "--help"}, expected: []string{"bookist lists add-book - Add a book to a list", "--book string", "--list string"}},
		{name: "reads list", args: []string{"reads", "list", "--help"}, expected: []string{"bookist reads list - List reads for a book", "--book string", "--format string"}},
		{name: "reads add", args: []string{"reads", "add", "--help"}, expected: []string{"bookist reads add - Record a read for a book", "--book string", "--rating float"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			exitCode, stdout, stderr := runCLI(test.args)

			if exitCode != 0 {
				t.Fatalf("expected exit code 0, got %d", exitCode)
			}
			if stderr != "" {
				t.Fatalf("expected empty stderr, got %q", stderr)
			}
			for _, expected := range test.expected {
				if !strings.Contains(stdout, expected) {
					t.Errorf("expected stdout to contain %q, got:\n%s", expected, stdout)
				}
			}
		})
	}
}

// ── List Format Validation ────────────────────────────────────────────────────

func TestListCommandsRejectUnsupportedFormat(t *testing.T) {
	for _, resource := range []string{"books", "authors", "lists", "reads"} {
		t.Run(resource, func(t *testing.T) {
			requestCount := 0
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				requestCount++
			}))
			defer server.Close()

			exitCode, stdout, stderr := runCLI([]string{resource, "list", "--server", server.URL, "--format", "xml"})
			if exitCode != 2 {
				t.Fatalf("expected exit code 2, got %d", exitCode)
			}
			if stdout != "" {
				t.Fatalf("expected empty stdout, got %q", stdout)
			}
			expected := `Error: unsupported output format "xml": must be one of tsv, pretty, json`
			if !strings.Contains(stderr, expected) {
				t.Fatalf("expected stderr to contain %q, got %q", expected, stderr)
			}
			if requestCount != 0 {
				t.Fatalf("expected format validation before HTTP request, got %d requests", requestCount)
			}
		})
	}
}

// ── Error Help ────────────────────────────────────────────────────────────────

func TestFlagErrorShowsOnlyCommandHelp(t *testing.T) {
	exitCode, stdout, stderr := runCLI([]string{"books", "add", "--pages", "many"})

	if exitCode != 2 {
		t.Fatalf("expected exit code 2, got %d", exitCode)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, `invalid value "many" for flag -pages`) {
		t.Errorf("expected invalid flag error, got:\n%s", stderr)
	}
	if !strings.Contains(stderr, "bookist books add - Add a book") {
		t.Errorf("expected books add help, got:\n%s", stderr)
	}
	if strings.Contains(stderr, "Usage of books add:") {
		t.Errorf("expected Go flag usage to be suppressed, got:\n%s", stderr)
	}
}

func TestInvalidSubcommandShowsParentHelp(t *testing.T) {
	for _, parent := range []string{"books", "authors", "lists", "reads"} {
		t.Run(parent, func(t *testing.T) {
			exitCode, stdout, stderr := runCLI([]string{parent, "invalid"})

			if exitCode != 2 {
				t.Fatalf("expected exit code 2, got %d", exitCode)
			}
			if stdout != "" {
				t.Fatalf("expected empty stdout, got %q", stdout)
			}
			if !strings.Contains(stderr, "Error: unknown "+parent+" command \"invalid\"") {
				t.Errorf("expected unknown command error, got:\n%s", stderr)
			}
			if !strings.Contains(stderr, "bookist "+parent+" - Manage") {
				t.Errorf("expected %s help, got:\n%s", parent, stderr)
			}
			if strings.Contains(stderr, "Start the Bookist server") {
				t.Errorf("expected parent help rather than root help, got:\n%s", stderr)
			}
		})
	}
}

// ── Help Command ──────────────────────────────────────────────────────────────

func TestHelpCommandAcceptsACommandPath(t *testing.T) {
	exitCode, stdout, stderr := runCLI([]string{"help", "books", "add"})

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d", exitCode)
	}
	if stderr != "" {
		t.Fatalf("expected empty stderr, got %q", stderr)
	}
	if !strings.Contains(stdout, "bookist books add - Add a book") {
		t.Fatalf("expected books add help, got:\n%s", stdout)
	}
}

func runCLI(args []string) (int, string, string) {
	var stdout, stderr strings.Builder
	exitCode := cli.Run(args, &stdout, &stderr)
	return exitCode, stdout.String(), stderr.String()
}
