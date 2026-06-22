package cli

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"bakku.dev/bookist/internal/books"
)

func runBooks(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		fmt.Fprintln(stderr, "missing books command")
		return 2
	}

	switch args[0] {
	case "list":
		return runBooksList(args[1:], stdout, stderr)
	case "add":
		return runBooksAdd(args[1:], stdout, stderr)
	default:
		fmt.Fprintf(stderr, "unknown books command %q\n", args[0])
		return 2
	}
}

func runBooksList(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("books list", flag.ContinueOnError)
	flags.SetOutput(stderr)
	serverURL := flags.String("server", defaultServerURL, "Bookist server URL")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	endpoint, err := joinURL(*serverURL, "/api/books")
	if err != nil {
		fmt.Fprintf(stderr, "invalid server URL: %v\n", err)
		return 2
	}

	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(endpoint)
	if err != nil {
		fmt.Fprintf(stderr, "list books: %v\n", err)
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		fmt.Fprintf(stderr, "list books: server returned %s\n", resp.Status)
		return 1
	}

	var listedBooks []books.Book
	if err := json.NewDecoder(resp.Body).Decode(&listedBooks); err != nil {
		fmt.Fprintf(stderr, "decode books: %v\n", err)
		return 1
	}

	for _, book := range listedBooks {
		isbn := ""
		if book.ISBN != nil {
			isbn = *book.ISBN
		}
		fmt.Fprintf(stdout, "%d\t%s\t%s\n", book.ID, book.Title, isbn)
	}

	return 0
}

func runBooksAdd(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("books add", flag.ContinueOnError)
	flags.SetOutput(stderr)
	serverURL := flags.String("server", defaultServerURL, "Bookist server URL")
	title := flags.String("title", "", "Book title")
	isbn := flags.String("isbn", "", "Book ISBN")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	input := books.CreateBookInput{Title: *title}
	if strings.TrimSpace(*isbn) != "" {
		input.ISBN = isbn
	}

	body, err := json.Marshal(input)
	if err != nil {
		fmt.Fprintf(stderr, "encode book: %v\n", err)
		return 1
	}

	endpoint, err := joinURL(*serverURL, "/api/books")
	if err != nil {
		fmt.Fprintf(stderr, "invalid server URL: %v\n", err)
		return 2
	}

	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		fmt.Fprintf(stderr, "add book: %v\n", err)
		return 1
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		fmt.Fprintf(stderr, "add book: server returned %s\n", resp.Status)
		return 1
	}

	var book books.Book
	if err := json.NewDecoder(resp.Body).Decode(&book); err != nil {
		fmt.Fprintf(stderr, "decode book: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "%d\t%s\n", book.ID, book.Title)
	return 0
}

func joinURL(base string, path string) (string, error) {
	parsed, err := url.Parse(base)
	if err != nil {
		return "", err
	}
	parsed.Path = strings.TrimRight(parsed.Path, "/") + path
	return parsed.String(), nil
}
