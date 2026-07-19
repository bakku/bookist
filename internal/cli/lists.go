package cli

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bakku.dev/bookist/internal/lists"
)

func runLists(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "Error: missing lists command")
		_, _ = fmt.Fprintln(stderr)
		printListsHelp(stderr)
		return 2
	}

	switch args[0] {
	case "ls":
		return runListsLS(args[1:], stdout, stderr)

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
			{name: "ls", description: "List book lists"},
			{name: "add", description: "Add a book list"},
			{name: "add-book", description: "Add a book to a list"},
		},
	}, nil)
}

func runListsLS(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("lists ls", flag.ContinueOnError)

	serverURL := flags.String("server", defaultServerURL, "Bookist server URL")
	formatValue := flags.String("format", string(outputFormatPretty), "Output format (tsv|pretty|json)")
	query := flags.String("query", "", "Filter lists by name")

	help := commandHelp{
		name:        "bookist lists ls",
		usage:       "bookist lists ls [options]",
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

	listed, err := fetchLists(*serverURL, *query)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ls lists: %v\n", err)
		return 1
	}

	rows := make([][]string, 0, len(listed))
	for _, list := range listed {
		rows = append(rows, []string{strconv.FormatInt(list.ID, 10), list.Name})
	}

	if err := writeListOutput(stdout, format, listed, []string{"ID", "NAME"}, rows); err != nil {
		_, _ = fmt.Fprintf(stderr, "ls lists: write output: %v\n", err)
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

	_, _ = fmt.Fprintf(stdout, "%d\t%s\n", created.ID, created.Name)
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

	endpoint, err := joinURL(*serverURL, "/api/lists/"+strconv.FormatInt(listID, 10)+"/books")
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

	_, _ = fmt.Fprintf(stdout, "added book %d to list %d\n", bookID, listID)
	return 0
}

func resolveListID(serverURL, value string) (int64, error) {
	value = strings.TrimSpace(value)

	id, isID, err := parseIDReference(value)
	if err != nil {
		return 0, err
	}
	if isID {
		return id, nil
	}

	existing, err := fetchLists(serverURL, value)
	if err != nil {
		return 0, fmt.Errorf("fetch lists: %v", err)
	}

	var matches []lists.List
	for _, l := range existing {
		if strings.EqualFold(l.Name, value) {
			matches = append(matches, l)
		}
	}

	if len(matches) > 1 {
		return 0, fmt.Errorf("list %q exists multiple times; pass a list ID instead", value)
	}
	if len(matches) == 1 {
		return matches[0].ID, nil
	}

	return 0, fmt.Errorf("list not found: %s", value)
}

func resolveBookID(serverURL, value string) (int64, error) {
	value = strings.TrimSpace(value)

	id, isID, err := parseIDReference(value)
	if err != nil {
		return 0, err
	}
	if isID {
		return id, nil
	}

	existing, err := fetchBooks(serverURL, value)
	if err != nil {
		return 0, fmt.Errorf("fetch books: %v", err)
	}

	byTitle := make(map[string][]int64)
	for _, b := range existing {
		if strings.EqualFold(b.Title, value) {
			key := strings.ToLower(b.Title)
			byTitle[key] = append(byTitle[key], b.ID)
		}
	}

	matches := byTitle[strings.ToLower(value)]
	if len(matches) > 1 {
		return 0, fmt.Errorf("book %q exists multiple times; pass a book ID instead", value)
	}
	if len(matches) == 1 {
		return matches[0], nil
	}

	return 0, fmt.Errorf("book not found: %s", value)
}

func fetchLists(serverURL, query string) ([]lists.List, error) {
	endpoint, err := joinURLWithQuery(serverURL, "/api/lists", query)
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
