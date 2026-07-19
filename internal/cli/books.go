package cli

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/books"
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
	formatValue := flags.String("format", string(outputFormatPretty), "Output format (tsv|pretty|json)")

	help := commandHelp{
		name:        "bookist books list",
		usage:       "bookist books list [options]",
		description: "List books",
	}
	if ok, exitCode := parseFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
	}

	format, err := parseOutputFormat(*formatValue)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}

	listedBooks, err := fetchBooks(*serverURL)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "list books: %v\n", err)
		return 1
	}

	rows := make([][]string, 0, len(listedBooks))
	for _, book := range listedBooks {
		isbn := ""
		if book.ISBN != nil {
			isbn = *book.ISBN
		}
		rows = append(rows, []string{strconv.FormatInt(book.ID, 10), book.Title, isbn})
	}

	if err := writeListOutput(stdout, format, listedBooks, []string{"ID", "TITLE", "ISBN"}, rows); err != nil {
		_, _ = fmt.Fprintf(stderr, "list books: write output: %v\n", err)
		return 1
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
	var purchasePrice optionalStringFlag
	var notes optionalStringFlag
	var summary optionalStringFlag
	var seriesName optionalStringFlag
	var seriesPosition optionalFloatFlag
	var location optionalStringFlag
	var condition optionalStringFlag
	var acquisitionSource optionalStringFlag
	var pages optionalIntFlag
	var publishedYear optionalIntFlag
	var publishedMonth optionalIntFlag
	var publishedDay optionalIntFlag

	flags.Var(&isbn, "isbn", "Book ISBN")
	flags.Var(&language, "language", "Book language")
	flags.Var(&publisher, "publisher", "Book publisher")
	flags.Var(&edition, "edition", "Book edition")
	flags.Var(&format, "format", "Book format (hardback|paperback|epub)")
	flags.Var(&purchasedAt, "purchased-at", "Date purchased (YYYY-MM-DD)")
	flags.Var(&purchasePrice, "purchase-price", "Book purchase price (free-form text)")
	flags.Var(&notes, "notes", "Personal notes")
	flags.Var(&summary, "summary", "Book summary")
	flags.Var(&seriesName, "series-name", "Book series name")
	flags.Var(&seriesPosition, "series-position", "Book position in its series")
	flags.Var(&location, "location", "Book storage location")
	flags.Var(&condition, "condition", "Book condition (new|very_good|good|acceptable|poor)")
	flags.Var(&acquisitionSource, "acquisition-source", "Book acquisition source")
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
		Title:             *title,
		ISBN:              isbn.value,
		Language:          language.value,
		Publisher:         publisher.value,
		Edition:           edition.value,
		PurchasedAt:       purchasedAt.value,
		PurchasePrice:     purchasePrice.value,
		Pages:             pages.value,
		Notes:             notes.value,
		Summary:           summary.value,
		SeriesName:        seriesName.value,
		SeriesPosition:    seriesPosition.value,
		Location:          location.value,
		AcquisitionSource: acquisitionSource.value,
		PublishedYear:     publishedYear.value,
		PublishedMonth:    publishedMonth.value,
		PublishedDay:      publishedDay.value,
	}

	if format.value != nil {
		f := books.Format(*format.value)
		input.Format = &f
	}

	if condition.value != nil {
		c := books.Condition(*condition.value)
		input.Condition = &c
	}

	if len(authorFlags) > 0 {
		authorIDs, err := resolveAuthorIDs(*serverURL, authorFlags)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "%v\n", err)
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

	_, _ = fmt.Fprintf(stdout, "%d\t%s\n", book.ID, book.Title)
	return 0
}

func resolveAuthorIDs(serverURL string, values []string) ([]int64, error) {
	if len(values) == 0 {
		return nil, nil
	}

	var result []int64
	var byName map[string][]authors.Author

	for _, val := range values {
		val = strings.TrimSpace(val)
		if val == "" {
			continue
		}

		id, isID, err := parseIDReference(val)
		if err != nil {
			return nil, err
		}

		if isID {
			result = append(result, id)
		} else {
			if byName == nil {
				existingAuthors, err := fetchAuthors(serverURL)
				if err != nil {
					return nil, fmt.Errorf("fetch authors: %v", err)
				}

				byName = make(map[string][]authors.Author, len(existingAuthors))

				for _, a := range existingAuthors {
					key := strings.ToLower(a.Name)
					byName[key] = append(byName[key], a)
				}
			}

			key := strings.ToLower(val)
			matches := byName[key]
			if len(matches) > 1 {
				return nil, fmt.Errorf("author %q exists multiple times; pass an author ID instead", val)
			}
			if len(matches) == 1 {
				result = append(result, matches[0].ID)
			} else {
				// Textual author references create the missing author for convenient book entry.
				created, err := createAuthor(serverURL, val)
				if err != nil {
					return nil, fmt.Errorf("create author %q: %v", val, err)
				}

				result = append(result, created.ID)
				byName[key] = []authors.Author{created}
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
