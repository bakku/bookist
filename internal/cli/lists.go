package cli

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"bakku.dev/bookist/internal/lists"
	"github.com/google/uuid"
)

func runLists(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "Error: missing lists command")
		_, _ = fmt.Fprintln(stderr)
		printListsHelp(stderr)
		return 2
	}

	switch args[0] {
	case "list":
		return runListsList(args[1:], stdout, stderr)

	case "add":
		return runListsAdd(args[1:], stdout, stderr)

	case "add-book":
		return runListsAddBook(args[1:], stdout, stderr)

	case "help", "-h", "--help":
		printListsHelp(stdout)
		return 0

	default:
		_, _ = fmt.Fprintf(stderr, "Error: unknown lists command %q\n\n", args[0])
		printListsHelp(stderr)
		return 2
	}
}

func printListsHelp(w io.Writer) {
	printCommandHelp(w, commandHelp{
		name:        "bookist lists",
		usage:       "bookist lists [command [command options]]",
		description: "Manage book lists",
		commands: []helpCommand{
			{name: "list", description: "List book lists"},
			{name: "add", description: "Add a book list"},
			{name: "add-book", description: "Add a book to a list"},
		},
	}, nil)
}

func runListsList(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("lists list", flag.ContinueOnError)

	serverURL := flags.String("server", defaultServerURL, "Bookist server URL")
	formatValue := flags.String("format", string(outputFormatPretty), "Output format (tsv|pretty|json)")

	help := commandHelp{
		name:        "bookist lists list",
		usage:       "bookist lists list [options]",
		description: "List book lists",
	}
	if ok, exitCode := parseFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
	}

	format, err := parseOutputFormat(*formatValue)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}

	listed, err := fetchLists(*serverURL)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "list lists: %v\n", err)
		return 1
	}

	rows := make([][]string, 0, len(listed))
	for _, list := range listed {
		rows = append(rows, []string{list.ID, list.Name})
	}

	if err := writeListOutput(stdout, format, listed, []string{"ID", "NAME"}, rows); err != nil {
		_, _ = fmt.Fprintf(stderr, "list lists: write output: %v\n", err)
		return 1
	}

	return 0
}

func runListsAdd(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("lists add", flag.ContinueOnError)

	serverURL := flags.String("server", defaultServerURL, "Bookist server URL")
	name := flags.String("name", "", "List name")
	description := flags.String("description", "", "List description")

	help := commandHelp{
		name:        "bookist lists add",
		usage:       "bookist lists add [options]",
		description: "Add a book list",
	}
	if ok, exitCode := parseFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
	}

	trimmedName := strings.TrimSpace(*name)

	input := lists.CreateListRequest{Name: trimmedName}

	if strings.TrimSpace(*description) != "" {
		input.Description = description
	}

	body, err := json.Marshal(input)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "encode list: %v\n", err)
		return 1
	}

	endpoint, err := joinURL(*serverURL, "/api/lists")
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "invalid server URL: %v\n", err)
		return 2
	}

	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "add list: %v\n", err)
		return 1
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusCreated {
		_, _ = fmt.Fprintf(stderr, "add list: server returned %s\n", resp.Status)
		return 1
	}

	var created lists.List
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		_, _ = fmt.Fprintf(stderr, "decode list: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "%s\t%s\n", created.ID, created.Name)
	return 0
}

func runListsAddBook(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("lists add-book", flag.ContinueOnError)

	serverURL := flags.String("server", defaultServerURL, "Bookist server URL")
	listRef := flags.String("list", "", "List name or ID")
	bookRef := flags.String("book", "", "Book title or ID")

	help := commandHelp{
		name:        "bookist lists add-book",
		usage:       "bookist lists add-book [options]",
		description: "Add a book to a list",
	}
	if ok, exitCode := parseFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
	}

	if strings.TrimSpace(*listRef) == "" {
		_, _ = fmt.Fprintln(stderr, "--list is required")
		return 2
	}

	if strings.TrimSpace(*bookRef) == "" {
		_, _ = fmt.Fprintln(stderr, "--book is required")
		return 2
	}

	listID, err := resolveListID(*serverURL, *listRef)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	bookID, err := resolveBookID(*serverURL, *bookRef)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	endpoint, err := joinURL(*serverURL, "/api/lists/"+listID+"/books")
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "invalid server URL: %v\n", err)
		return 2
	}

	input := lists.AddBookToListRequest{BookID: bookID}
	body, err := json.Marshal(input)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "encode request: %v\n", err)
		return 1
	}

	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "add book to list: %v\n", err)
		return 1
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusNoContent {
		_, _ = fmt.Fprintf(stderr, "add book to list: server returned %s\n", resp.Status)
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "added book %s to list %s\n", bookID, listID)
	return 0
}

func resolveListID(serverURL, value string) (string, error) {
	value = strings.TrimSpace(value)

	parsed, err := uuid.Parse(value)
	if err == nil && parsed.String() == strings.ToLower(value) {
		return parsed.String(), nil
	}

	existing, err := fetchLists(serverURL)
	if err != nil {
		return "", fmt.Errorf("fetch lists: %v", err)
	}

	byName := make(map[string]string)
	for _, l := range existing {
		if _, exists := byName[strings.ToLower(l.Name)]; !exists {
			byName[strings.ToLower(l.Name)] = l.ID
		}
	}

	if id, ok := byName[strings.ToLower(value)]; ok {
		return id, nil
	}

	return "", fmt.Errorf("list not found: %s", value)
}

func resolveBookID(serverURL, value string) (string, error) {
	value = strings.TrimSpace(value)

	parsed, err := uuid.Parse(value)
	if err == nil && parsed.String() == strings.ToLower(value) {
		return parsed.String(), nil
	}

	existing, err := fetchBooks(serverURL)
	if err != nil {
		return "", fmt.Errorf("fetch books: %v", err)
	}

	byTitle := make(map[string]string)
	for _, b := range existing {
		if _, exists := byTitle[strings.ToLower(b.Title)]; !exists {
			byTitle[strings.ToLower(b.Title)] = b.ID
		}
	}

	if id, ok := byTitle[strings.ToLower(value)]; ok {
		return id, nil
	}

	return "", fmt.Errorf("book not found: %s", value)
}

func fetchLists(serverURL string) ([]lists.List, error) {
	endpoint, err := joinURL(serverURL, "/api/lists")
	if err != nil {
		return nil, fmt.Errorf("invalid server URL: %v", err)
	}

	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("fetch lists: %v", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch lists: server returned %s", resp.Status)
	}

	var listed []lists.List
	if err := json.NewDecoder(resp.Body).Decode(&listed); err != nil {
		return nil, fmt.Errorf("decode lists: %v", err)
	}

	return listed, nil
}
