package cli

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"time"

	"bakku.dev/bookist/internal/authors"
)

func fetchAuthors(serverURL string) ([]authors.Author, error) {
	endpoint, err := joinURL(serverURL, "/api/authors")
	if err != nil {
		return nil, fmt.Errorf("invalid server URL: %v", err)
	}

	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("fetch authors: %v", err)
	}
	defer resp.Body.Close()

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
	defer resp.Body.Close()

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
		fmt.Fprintln(stderr, "missing authors command")
		return 2
	}

	switch args[0] {
	case "list":
		return runAuthorsList(args[1:], stdout, stderr)

	case "add":
		return runAuthorsAdd(args[1:], stdout, stderr)

	default:
		fmt.Fprintf(stderr, "unknown authors command %q\n", args[0])
		return 2
	}
}

func runAuthorsList(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("authors list", flag.ContinueOnError)
	serverURL := flags.String("server", defaultServerURL, "Bookist server URL")

	flags.SetOutput(stderr)

	if err := flags.Parse(args); err != nil {
		return 2
	}

	listed, err := fetchAuthors(*serverURL)
	if err != nil {
		fmt.Fprintf(stderr, "list authors: %v\n", err)
		return 1
	}

	for _, author := range listed {
		fmt.Fprintf(stdout, "%s\t%s\n", author.ID, author.Name)
	}

	return 0
}

func runAuthorsAdd(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("authors add", flag.ContinueOnError)
	serverURL := flags.String("server", defaultServerURL, "Bookist server URL")
	name := flags.String("name", "", "Author name")

	flags.SetOutput(stderr)

	if err := flags.Parse(args); err != nil {
		return 2
	}

	author, err := createAuthor(*serverURL, *name)
	if err != nil {
		fmt.Fprintf(stderr, "add author: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "%s\t%s\n", author.ID, author.Name)
	return 0
}
