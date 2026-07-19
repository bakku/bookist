package cli

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"time"

	"bakku.dev/bookist/internal/authors"
)

func fetchAuthors(serverURL, query string) ([]authors.Author, error) {
	endpoint, err := joinURLWithQuery(serverURL, "/api/authors", query)
	if err != nil {
		return nil, fmt.Errorf("invalid server URL: %v", err)
	}

	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("fetch authors: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch authors: server returned %s", resp.Status)
	}

	var listed []authors.Author
	if err := json.NewDecoder(resp.Body).Decode(&listed); err != nil {
		return nil, fmt.Errorf("decode authors: %v", err)
	}

	return listed, nil
}

func createAuthor(serverURL, name string) (authors.Author, error) {
	input := authors.CreateAuthorRequest{Name: name}
	body, err := json.Marshal(input)
	if err != nil {
		return authors.Author{}, err
	}

	endpoint, err := joinURL(serverURL, "/api/authors")
	if err != nil {
		return authors.Author{}, fmt.Errorf("invalid server URL: %v", err)
	}

	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		return authors.Author{}, err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusCreated {
		return authors.Author{}, fmt.Errorf("server returned %s", resp.Status)
	}

	var author authors.Author
	if err := json.NewDecoder(resp.Body).Decode(&author); err != nil {
		return authors.Author{}, err
	}

	return author, nil
}

func runAuthors(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "Error: missing authors command")
		_, _ = fmt.Fprintln(stderr)
		printAuthorsHelp(stderr)
		return 2
	}

	switch args[0] {
	case "ls":
		return runAuthorsLS(args[1:], stdout, stderr)

	case "add":
		return runAuthorsAdd(args[1:], stdout, stderr)

	case "help", "-h", "--help":
		printAuthorsHelp(stdout)
		return 0

	default:
		_, _ = fmt.Fprintf(stderr, "Error: unknown authors command %q\n\n", args[0])
		printAuthorsHelp(stderr)
		return 2
	}
}

func printAuthorsHelp(w io.Writer) {
	printCommandHelp(w, commandHelp{
		name:        "bookist authors",
		usage:       "bookist authors [command [command options]]",
		description: "Manage authors",
		commands: []helpCommand{
			{name: "ls", description: "List authors"},
			{name: "add", description: "Add an author"},
		},
	}, nil)
}

func runAuthorsLS(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("authors ls", flag.ContinueOnError)

	serverURL := flags.String("server", defaultServerURL, "Bookist server URL")
	formatValue := flags.String("format", string(outputFormatPretty), "Output format (tsv|pretty|json)")
	query := flags.String("query", "", "Filter authors by name")

	help := commandHelp{
		name:        "bookist authors ls",
		usage:       "bookist authors ls [options]",
		description: "List authors",
	}
	if ok, exitCode := parseFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
	}

	format, err := parseOutputFormat(*formatValue)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}

	listed, err := fetchAuthors(*serverURL, *query)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ls authors: %v\n", err)
		return 1
	}

	rows := make([][]string, 0, len(listed))
	for _, author := range listed {
		rows = append(rows, []string{strconv.FormatInt(author.ID, 10), author.Name})
	}

	if err := writeListOutput(stdout, format, listed, []string{"ID", "NAME"}, rows); err != nil {
		_, _ = fmt.Fprintf(stderr, "ls authors: write output: %v\n", err)
		return 1
	}

	return 0
}

func runAuthorsAdd(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("authors add", flag.ContinueOnError)

	serverURL := flags.String("server", defaultServerURL, "Bookist server URL")
	name := flags.String("name", "", "Author name")

	help := commandHelp{
		name:        "bookist authors add",
		usage:       "bookist authors add [options]",
		description: "Add an author",
	}
	if ok, exitCode := parseFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
	}

	author, err := createAuthor(*serverURL, *name)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "add author: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "%d\t%s\n", author.ID, author.Name)
	return 0
}
