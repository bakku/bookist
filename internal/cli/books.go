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

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/books"
	"github.com/google/uuid"
)

func runBooks(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "Error: missing books command")
		_, _ = fmt.Fprintln(stderr)
		printBooksHelp(stderr)
		return 2
	}

	switch args[0] {
	case "list":
		return runBooksList(args[1:], stdout, stderr)

	case "add":
		return runBooksAdd(args[1:], stdout, stderr)

	case "help", "-h", "--help":
		printBooksHelp(stdout)
		return 0

	default:
		_, _ = fmt.Fprintf(stderr, "Error: unknown books command %q\n\n", args[0])
		printBooksHelp(stderr)
		return 2
	}
}

func printBooksHelp(w io.Writer) {
	printCommandHelp(w, commandHelp{
		name:        "bookist books",
		usage:       "bookist books [command [command options]]",
		description: "Manage books",
		commands: []helpCommand{
			{name: "list", description: "List books"},
			{name: "add", description: "Add a book"},
		},
	}, nil)
}

func runBooksList(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("books list", flag.ContinueOnError)
	flags.SetOutput(stderr)

	serverURL := flags.String("server", defaultServerURL, "Bookist server URL")

	help := commandHelp{
		name:        "bookist books list",
		usage:       "bookist books list [options]",
		description: "List books",
	}
	if ok, exitCode := parseFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
	}

	listedBooks, err := fetchBooks(*serverURL)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "list books: %v\n", err)
		return 1
	}

	for _, book := range listedBooks {
		isbn := ""
		if book.ISBN != nil {
			isbn = *book.ISBN
		}
		_, _ = fmt.Fprintf(stdout, "%s\t%s\t%s\n", book.ID, book.Title, isbn)
	}

	return 0
}

func fetchBooks(serverURL string) ([]books.Book, error) {
	endpoint, err := joinURL(serverURL, "/api/books")
	if err != nil {
		return nil, fmt.Errorf("invalid server URL: %v", err)
	}

	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("fetch books: %v", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch books: server returned %s", resp.Status)
	}

	var listed []books.Book
	if err := json.NewDecoder(resp.Body).Decode(&listed); err != nil {
		return nil, fmt.Errorf("decode books: %v", err)
	}

	return listed, nil
}

func runBooksAdd(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("books add", flag.ContinueOnError)

	serverURL := flags.String("server", defaultServerURL, "Bookist server URL")
	title := flags.String("title", "", "Book title")

	var authorFlags stringSliceFlag
	flags.Var(&authorFlags, "author", "Author name or ID (repeatable)")

	var isbn optionalStringFlag
	var language optionalStringFlag
	var publisher optionalStringFlag
	var edition optionalStringFlag
	var format optionalStringFlag
	var purchasedAt optionalStringFlag
	var notes optionalStringFlag
	var pages optionalIntFlag
	var publishedYear optionalIntFlag
	var publishedMonth optionalIntFlag
	var publishedDay optionalIntFlag

	flags.Var(&isbn, "isbn", "Book ISBN")
	flags.Var(&language, "language", "Book language")
	flags.Var(&publisher, "publisher", "Book publisher")
	flags.Var(&edition, "edition", "Book edition")
	flags.Var(&format, "format", "Book format (hardback|paperback|epub)")
	flags.Var(&purchasedAt, "purchased-at", "Date purchased (ISO 8601)")
	flags.Var(&notes, "notes", "Personal notes")
	flags.Var(&pages, "pages", "Number of pages")
	flags.Var(&publishedYear, "published-year", "Publication year")
	flags.Var(&publishedMonth, "published-month", "Publication month (1-12)")
	flags.Var(&publishedDay, "published-day", "Publication day (1-31)")

	help := commandHelp{
		name:        "bookist books add",
		usage:       "bookist books add [options]",
		description: "Add a book",
	}
	if ok, exitCode := parseFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
	}

	input := books.CreateBookRequest{
		Title:          *title,
		ISBN:           isbn.value,
		Language:       language.value,
		Publisher:      publisher.value,
		Edition:        edition.value,
		PurchasedAt:    purchasedAt.value,
		Pages:          pages.value,
		Notes:          notes.value,
		PublishedYear:  publishedYear.value,
		PublishedMonth: publishedMonth.value,
		PublishedDay:   publishedDay.value,
	}

	if format.value != nil {
		f := books.Format(*format.value)
		input.Format = &f
	}

	if len(authorFlags) > 0 {
		authorIDs, err := resolveAuthorIDs(*serverURL, authorFlags)
		if err != nil {
			fmt.Fprintf(stderr, "%v\n", err)
			return 1
		}
		input.AuthorIDs = authorIDs
	}

	body, err := json.Marshal(input)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "encode book: %v\n", err)
		return 1
	}

	endpoint, err := joinURL(*serverURL, "/api/books")
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "invalid server URL: %v\n", err)
		return 2
	}

	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "add book: %v\n", err)
		return 1
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusCreated {
		_, _ = fmt.Fprintf(stderr, "add book: server returned %s\n", resp.Status)
		return 1
	}

	var book books.Book
	if err := json.NewDecoder(resp.Body).Decode(&book); err != nil {
		_, _ = fmt.Fprintf(stderr, "decode book: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "%s\t%s\n", book.ID, book.Title)
	return 0
}

func resolveAuthorIDs(serverURL string, values []string) ([]string, error) {
	if len(values) == 0 {
		return nil, nil
	}

	existingAuthors, err := fetchAuthors(serverURL)
	if err != nil {
		return nil, fmt.Errorf("fetch authors: %v", err)
	}

	byID := make(map[string]authors.Author)
	byName := make(map[string]authors.Author)

	for _, a := range existingAuthors {
		byID[a.ID] = a

		if _, exists := byName[a.Name]; !exists {
			byName[a.Name] = a
		}
	}

	var result []string

	for _, val := range values {
		val = strings.TrimSpace(val)
		if val == "" {
			continue
		}

		parsed, err := uuid.Parse(val)
		if err == nil {
			id := parsed.String()

			if _, ok := byID[id]; ok {
				result = append(result, id)
			} else {
				return nil, fmt.Errorf("author not found: %s", id)
			}
		} else {
			if a, ok := byName[val]; ok {
				result = append(result, a.ID)
			} else {
				created, err := createAuthor(serverURL, val)
				if err != nil {
					return nil, fmt.Errorf("create author %q: %v", val, err)
				}
				result = append(result, created.ID)
			}
		}
	}

	return result, nil
}

func joinURL(base string, path string) (string, error) {
	parsed, err := url.Parse(base)
	if err != nil {
		return "", err
	}

	parsed.Path = strings.TrimRight(parsed.Path, "/") + path

	return parsed.String(), nil
}
