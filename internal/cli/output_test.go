package cli_test

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/cli"
	"bakku.dev/bookist/internal/lists"
	"bakku.dev/bookist/internal/reads"
)

// ── Server Environment Variable ──────────────────────────────────────────────

func TestServerDefaultsToEnvironmentVariable(t *testing.T) {
	var gotPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewEncoder(w).Encode([]authors.Author{})
	}))
	defer server.Close()
	setEnv(t, map[string]string{"BOOKIST_SERVER": server.URL})

	exitCode, _, stderr := runCLI([]string{"authors", "ls"})
	if exitCode != 0 || stderr != "" {
		t.Fatalf("unexpected result: exit=%d stderr=%q", exitCode, stderr)
	}
	if gotPath != "/api/authors" {
		t.Fatalf("expected request to the BOOKIST_SERVER server, got path %q", gotPath)
	}
}

func TestServerFlagOverridesEnvironmentVariable(t *testing.T) {
	envServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("unexpected request to BOOKIST_SERVER server: %s %s", r.Method, r.URL.Path)
	}))
	defer envServer.Close()

	var gotPath string
	flagServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		_ = json.NewEncoder(w).Encode([]authors.Author{})
	}))
	defer flagServer.Close()
	setEnv(t, map[string]string{"BOOKIST_SERVER": envServer.URL})

	exitCode, _, stderr := runCLI([]string{"authors", "ls", "--server", flagServer.URL})
	if exitCode != 0 || stderr != "" {
		t.Fatalf("unexpected result: exit=%d stderr=%q", exitCode, stderr)
	}
	if gotPath != "/api/authors" {
		t.Fatalf("expected request to the --server server, got path %q", gotPath)
	}
}

func TestInvalidEnvironmentServerURLIsRejected(t *testing.T) {
	setEnv(t, map[string]string{"BOOKIST_SERVER": "localhost:8080"})

	exitCode, stdout, stderr := runCLI([]string{"authors", "ls"})
	if exitCode != 2 || stdout != "" || !strings.Contains(stderr, "invalid server URL") {
		t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
}

func TestServerHelpShowsEnvironmentDefault(t *testing.T) {
	tests := []struct {
		name     string
		env      map[string]string
		expected string
	}{
		{name: "unset", expected: "(default: http://localhost:8080)"},
		{name: "blank", env: map[string]string{"BOOKIST_SERVER": "  "}, expected: "(default: http://localhost:8080)"},
		{name: "set", env: map[string]string{"BOOKIST_SERVER": "http://library:9000"}, expected: "(default: http://library:9000)"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			setEnv(t, test.env)

			exitCode, stdout, stderr := runCLI([]string{"books", "ls", "--help"})
			if exitCode != 0 || stderr != "" {
				t.Fatalf("unexpected result: exit=%d stderr=%q", exitCode, stderr)
			}
			for _, expected := range []string{"BOOKIST_SERVER", test.expected} {
				if !strings.Contains(stdout, expected) {
					t.Errorf("expected stdout to contain %q, got:\n%s", expected, stdout)
				}
			}
		})
	}
}

// ── Output Flag ──────────────────────────────────────────────────────────────

func TestListAndShowAcceptOutputFlagAndShortAlias(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(serveReadOnlyRecords))
	defer server.Close()

	tests := []struct {
		name string
		args []string
	}{
		{name: "books ls", args: []string{"books", "ls"}},
		{name: "authors ls", args: []string{"authors", "ls"}},
		{name: "lists ls", args: []string{"lists", "ls"}},
		{name: "reads ls", args: []string{"reads", "ls", "--book", "id:1"}},
		{name: "books show", args: []string{"books", "show", "id:1"}},
		{name: "authors show", args: []string{"authors", "show", "id:1"}},
		{name: "lists show", args: []string{"lists", "show", "id:1"}},
		{name: "reads show", args: []string{"reads", "show", "1"}},
	}

	for _, test := range tests {
		for _, flag := range [][]string{{"-o", "json"}, {"--output", "json"}, {"-o=json"}, {"--output=json"}} {
			t.Run(test.name+" "+strings.Join(flag, " "), func(t *testing.T) {
				args := append(append([]string{}, test.args...), flag...)
				args = append(args, "--server", server.URL)

				exitCode, stdout, stderr := runCLI(args)
				if exitCode != 0 || stderr != "" {
					t.Fatalf("unexpected result: exit=%d stderr=%q", exitCode, stderr)
				}
				if !json.Valid([]byte(stdout)) {
					t.Fatalf("expected JSON output, got %q", stdout)
				}
			})
		}
	}
}

func TestShortOutputFlagSelectsTSV(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode([]authors.Author{{ID: 6, Name: "Ursula K. Le Guin"}})
	}))
	defer server.Close()

	exitCode, stdout, stderr := runCLI([]string{"authors", "ls", "-o", "tsv", "--server", server.URL})
	if exitCode != 0 || stderr != "" || stdout != "6\tUrsula K. Le Guin\n" {
		t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
}

func TestOutputFlagIsDocumentedWithShortAlias(t *testing.T) {
	for _, args := range [][]string{
		{"books", "ls", "--help"}, {"books", "show", "--help"}, {"books", "add", "--help"},
		{"authors", "rm", "--help"}, {"lists", "add-book", "--help"}, {"reads", "edit", "--help"},
	} {
		t.Run(strings.Join(args, " "), func(t *testing.T) {
			exitCode, stdout, stderr := runCLI(args)
			if exitCode != 0 || stderr != "" {
				t.Fatalf("unexpected result: exit=%d stderr=%q", exitCode, stderr)
			}
			if !strings.Contains(stdout, "--output, -o string") {
				t.Errorf("expected --output, -o in help, got:\n%s", stdout)
			}
			if strings.Contains(stdout, "--o ") {
				t.Errorf("expected the -o alias not to be listed separately, got:\n%s", stdout)
			}
		})
	}
}

func TestOldOutputFormatFlagIsRejected(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
	}))
	defer server.Close()

	for _, args := range [][]string{{"authors", "ls"}, {"authors", "show", "1"}, {"authors", "add", "--name", "Ada"}} {
		exitCode, stdout, stderr := runCLI(append(args, "--format", "json", "--server", server.URL))
		if exitCode != 2 || stdout != "" || !strings.Contains(stderr, "flag provided but not defined: -format") {
			t.Fatalf("%v: unexpected result: exit=%d stdout=%q stderr=%q", args, exitCode, stdout, stderr)
		}
	}
	if requestCount != 0 {
		t.Fatalf("expected no HTTP requests, got %d", requestCount)
	}
}

// ── Mutation Output ──────────────────────────────────────────────────────────

func TestMutationsPrintRecordOrResultAsJSON(t *testing.T) {
	tests := []struct {
		name   string
		args   []string
		status int
		record any
		want   any
	}{
		{
			name:   "books add",
			args:   []string{"books", "add", "--title", "Dune"},
			status: http.StatusCreated,
			record: books.Book{ID: 12, Title: "Dune", Authors: []authors.Author{{ID: 1, Name: "Frank Herbert"}}},
		},
		{
			name:   "books edit",
			args:   []string{"books", "edit", "id:12", "--title", "Dune Messiah"},
			status: http.StatusOK,
			record: books.Book{ID: 12, Title: "Dune Messiah"},
		},
		{name: "books rm", args: []string{"books", "rm", "id:12"}, want: map[string]any{"id": 12, "removed": true}},
		{
			name:   "authors add",
			args:   []string{"authors", "add", "--name", "Frank Herbert"},
			status: http.StatusCreated,
			record: authors.Author{ID: 6, Name: "Frank Herbert"},
		},
		{
			name:   "authors edit",
			args:   []string{"authors", "edit", "id:6", "--name", "Frank Herbert"},
			status: http.StatusOK,
			record: authors.Author{ID: 6, Name: "Frank Herbert"},
		},
		{name: "authors rm", args: []string{"authors", "rm", "id:6"}, want: map[string]any{"id": 6, "removed": true}},
		{
			name:   "lists add",
			args:   []string{"lists", "add", "--name", "Nightstand"},
			status: http.StatusCreated,
			record: lists.List{ID: 5, Name: "Nightstand"},
		},
		{
			name:   "lists edit",
			args:   []string{"lists", "edit", "id:5", "--name", "Nightstand"},
			status: http.StatusOK,
			record: lists.List{ID: 5, Name: "Nightstand"},
		},
		{name: "lists rm", args: []string{"lists", "rm", "id:5"}, want: map[string]any{"id": 5, "removed": true}},
		{
			name:   "lists add-book",
			args:   []string{"lists", "add-book", "--list", "id:5", "--book", "id:12"},
			status: http.StatusNoContent,
			want:   map[string]any{"list_id": 5, "book_id": 12, "added": true},
		},
		{
			name: "lists rm-book",
			args: []string{"lists", "rm-book", "--list", "id:5", "--book", "id:12"},
			want: map[string]any{"list_id": 5, "book_id": 12, "removed": true},
		},
		{
			name:   "reads add",
			args:   []string{"reads", "add", "--book", "id:12", "--rating", "4"},
			status: http.StatusCreated,
			record: reads.Read{ID: 9, BookID: 12, Rating: new(4.0)},
		},
		{
			name:   "reads edit",
			args:   []string{"reads", "edit", "9", "--rating", "4"},
			status: http.StatusOK,
			record: reads.Read{ID: 9, BookID: 12, Rating: new(4.0)},
		},
		{name: "reads rm", args: []string{"reads", "rm", "9"}, want: map[string]any{"id": 9, "removed": true}},
	}

	for _, test := range tests {
		for _, flag := range []string{"-o", "--output"} {
			t.Run(test.name+" "+flag, func(t *testing.T) {
				server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					serveMutation(w, r, test.status, test.record)
				}))
				defer server.Close()

				args := append(append([]string{}, test.args...), flag, "json", "--server", server.URL)
				exitCode, stdout, stderr := runCLI(args)
				if exitCode != 0 || stderr != "" {
					t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
				}
				if strings.Count(stdout, "\n") != 1 || !strings.HasSuffix(stdout, "\n") {
					t.Fatalf("expected a single JSON line, got %q", stdout)
				}

				want := test.want
				if test.record != nil {
					want = toJSONValue(t, test.record)
				}
				if got := toJSONValue(t, json.RawMessage(stdout)); !reflect.DeepEqual(got, toJSONValue(t, want)) {
					t.Fatalf("expected JSON %v, got %v", want, got)
				}
			})
		}
	}
}

func TestMutationsDefaultToTextOutput(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		serveMutation(w, r, http.StatusCreated, authors.Author{ID: 6, Name: "Frank Herbert"})
	}))
	defer server.Close()

	for _, format := range [][]string{nil, {"-o", "text"}, {"--output", "text"}} {
		args := append([]string{"authors", "add", "--name", "Frank Herbert", "--server", server.URL}, format...)

		exitCode, stdout, stderr := runCLI(args)
		if exitCode != 0 || stderr != "" || stdout != "6\tFrank Herbert\n" {
			t.Fatalf("%v: unexpected result: exit=%d stdout=%q stderr=%q", format, exitCode, stdout, stderr)
		}
	}
}

func TestMutationsRejectUnsupportedOutputBeforeRequest(t *testing.T) {
	requestCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestCount++
	}))
	defer server.Close()

	tests := [][]string{
		{"books", "add", "--title", "Dune"},
		{"books", "edit", "id:1", "--title", "Dune"},
		{"books", "rm", "id:1"},
		{"authors", "add", "--name", "Ada"},
		{"authors", "edit", "id:1", "--name", "Ada"},
		{"authors", "rm", "id:1"},
		{"lists", "add", "--name", "Owned"},
		{"lists", "edit", "id:1", "--name", "Owned"},
		{"lists", "rm", "id:1"},
		{"lists", "add-book", "--list", "id:1", "--book", "id:2"},
		{"lists", "rm-book", "--list", "id:1", "--book", "id:2"},
		{"reads", "add", "--book", "id:1"},
		{"reads", "edit", "1", "--rating", "4"},
		{"reads", "rm", "1"},
	}

	for _, args := range tests {
		t.Run(strings.Join(args[:2], " "), func(t *testing.T) {
			// pretty and tsv are list and show formats, not mutation formats.
			for _, format := range []string{"xml", "pretty"} {
				exitCode, stdout, stderr := runCLI(append(append([]string{}, args...), "-o", format, "--server", server.URL))
				expected := "Error: unsupported output format \"" + format + "\": must be one of text, json"
				if exitCode != 2 || stdout != "" || !strings.Contains(stderr, expected) {
					t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
				}
			}
		})
	}
	if requestCount != 0 {
		t.Fatalf("expected output validation before HTTP requests, got %d requests", requestCount)
	}
}

func TestBooksFormatFlagStillSetsBookFormat(t *testing.T) {
	var captured books.CreateBookRequest
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewDecoder(r.Body).Decode(&captured)
		w.WriteHeader(http.StatusCreated)
		_ = json.NewEncoder(w).Encode(books.Book{ID: 1, Title: "Dune"})
	}))
	defer server.Close()

	exitCode, stdout, stderr := runCLI([]string{"books", "add", "--title", "Dune", "--format", "epub", "-o", "json", "--server", server.URL})
	if exitCode != 0 || stderr != "" || !json.Valid([]byte(stdout)) {
		t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
	}
	if captured.Format == nil || *captured.Format != books.FormatEpub {
		t.Fatalf("expected book format epub, got %#v", captured.Format)
	}
}

// ── List Alias ───────────────────────────────────────────────────────────────

func TestListAliasBehavesLikeLS(t *testing.T) {
	for _, resource := range []string{"books", "authors", "lists", "reads"} {
		t.Run(resource, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(serveReadOnlyRecords))
			defer server.Close()

			args := []string{resource, "", "-o", "json", "--server", server.URL}
			if resource == "reads" {
				args = append(args, "--book", "id:1")
			}

			args[1] = "ls"
			lsExit, lsStdout, lsStderr := runCLI(args)
			args[1] = "list"
			listExit, listStdout, listStderr := runCLI(args)

			if lsExit != 0 || listExit != 0 || lsStderr != "" || listStderr != "" {
				t.Fatalf("unexpected result: ls=%d %q, list=%d %q", lsExit, lsStderr, listExit, listStderr)
			}
			if listStdout != lsStdout {
				t.Fatalf("expected list output %q to equal ls output %q", listStdout, lsStdout)
			}
		})
	}
}

func TestListAliasIsDocumentedInHelp(t *testing.T) {
	for _, resource := range []string{"books", "authors", "lists", "reads"} {
		t.Run(resource, func(t *testing.T) {
			exitCode, stdout, stderr := runCLI([]string{resource, "--help"})
			if exitCode != 0 || stderr != "" || !strings.Contains(stdout, "ls, list  ") {
				t.Fatalf("unexpected result: exit=%d stdout=%q stderr=%q", exitCode, stdout, stderr)
			}
		})
	}
}

// ── Helpers ──────────────────────────────────────────────────────────────────

// setEnv replaces the CLI's environment lookup with env for the test.
func setEnv(t *testing.T, env map[string]string) {
	t.Helper()

	cli.SetLookupEnv(t, func(key string) (string, bool) {
		value, ok := env[key]
		return value, ok
	})
}

// serveReadOnlyRecords answers GETs: collections are empty and a record path
// returns a record with the requested ID.
func serveReadOnlyRecords(w http.ResponseWriter, r *http.Request) {
	segments := strings.Split(strings.Trim(r.URL.Path, "/"), "/")
	last := segments[len(segments)-1]

	id, err := strconv.ParseInt(last, 10, 64)
	if err != nil {
		_ = json.NewEncoder(w).Encode([]any{})
		return
	}

	_ = json.NewEncoder(w).Encode(map[string]any{"id": id, "title": "Record", "name": "Record", "book_id": 1})
}

// serveMutation answers a mutation request with status and record, serving
// the GETs that resolve id:N references.
func serveMutation(w http.ResponseWriter, r *http.Request, status int, record any) {
	switch r.Method {
	case http.MethodGet:
		serveReadOnlyRecords(w, r)
	case http.MethodDelete:
		w.WriteHeader(http.StatusNoContent)
	default:
		w.WriteHeader(status)
		if record != nil {
			_ = json.NewEncoder(w).Encode(record)
		}
	}
}

// toJSONValue converts value to its generic JSON form for comparison.
func toJSONValue(t *testing.T, value any) any {
	t.Helper()

	encoded, err := json.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}

	var decoded any
	if err := json.Unmarshal(encoded, &decoded); err != nil {
		t.Fatal(err)
	}

	return decoded
}
