package cli

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/covers"
)

func runBooks(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "Error: missing books command")
		_, _ = fmt.Fprintln(stderr)
		printBooksHelp(stderr)
		return 2
	}

	switch args[0] {
	case "ls":
		return runBooksLS(args[1:], stdout, stderr)

	case "add":
		return runBooksAdd(args[1:], stdout, stderr)

	case "edit":
		return runBooksEdit(args[1:], stdout, stderr)

	case "rm":
		return runBooksRM(args[1:], stdout, stderr)

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
			{name: "ls", description: "List books"},
			{name: "add", description: "Add a book"},
			{name: "edit", description: "Edit a book"},
			{name: "rm", description: "Remove a book"},
		},
	}, nil)
}

func runBooksEdit(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("books edit", flag.ContinueOnError)
	serverURL := flags.String("server", defaultServerURL, "Bookist server URL")
	var title, cover optionalStringFlag
	var authorsFlag, clears stringSliceFlag
	var isbn, language, publisher, edition, format, purchasedAt, purchasePrice optionalStringFlag
	var notes, summary, seriesName, location, condition, acquisitionSource optionalStringFlag
	var seriesPosition optionalFloatFlag
	var pages, publishedYear, publishedMonth, publishedDay optionalIntFlag
	flags.Var(&title, "title", "Book title")
	flags.Var(&authorsFlag, "author", "Author name or ID (repeatable)")
	flags.Var(&cover, "cover", "Cover image file path or URL")
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
	flags.Var(&clears, "clear", "Field to clear (repeatable)")
	help := commandHelp{
		name:        "bookist books edit",
		usage:       "bookist books edit [options] <title-or-ID>",
		description: "Edit a book",
	}
	if ok, exitCode := parseFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
	}
	if flags.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "Error: books edit requires exactly one title or ID")
		_, _ = fmt.Fprintln(stderr)
		printCommandHelp(stderr, help, flags)
		return 2
	}

	changes := make(map[string]any)
	stringFields := []struct {
		key   string
		value *string
	}{
		{"title", title.value}, {"isbn", isbn.value}, {"language", language.value}, {"publisher", publisher.value},
		{"edition", edition.value}, {"format", format.value}, {"purchased_at", purchasedAt.value},
		{"purchase_price", purchasePrice.value}, {"notes", notes.value}, {"summary", summary.value},
		{"series_name", seriesName.value}, {"location", location.value}, {"condition", condition.value},
		{"acquisition_source", acquisitionSource.value},
	}
	for _, field := range stringFields {
		if field.value != nil {
			changes[field.key] = *field.value
		}
	}
	if seriesPosition.value != nil {
		changes["series_position"] = *seriesPosition.value
	}
	intFields := []struct {
		key   string
		value *int
	}{{"pages", pages.value}, {"published_year", publishedYear.value}, {"published_month", publishedMonth.value}, {"published_day", publishedDay.value}}
	for _, field := range intFields {
		if field.value != nil {
			changes[field.key] = *field.value
		}
	}
	if len(authorsFlag) > 0 {
		changes["author_ids"] = struct{}{}
	}
	if cover.value != nil {
		changes["cover"] = struct{}{}
	}
	clearable := map[string]string{
		"isbn": "isbn", "authors": "author_ids", "language": "language", "publisher": "publisher", "edition": "edition",
		"format": "format", "purchased-at": "purchased_at", "purchase-price": "purchase_price", "pages": "pages",
		"notes": "notes", "summary": "summary", "series-name": "series_name", "series-position": "series_position",
		"location": "location", "condition": "condition", "acquisition-source": "acquisition_source",
		"published-year": "published_year", "published-month": "published_month", "published-day": "published_day", "cover": "cover",
	}
	if err := validateClears(changes, clears, clearable); err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}
	if len(changes) == 0 {
		_, _ = fmt.Fprintln(stderr, "Error: books edit requires at least one change")
		return 2
	}
	for _, author := range authorsFlag {
		if strings.TrimSpace(author) == "" {
			_, _ = fmt.Fprintln(stderr, "Error: --author must not be blank; use --clear authors")
			return 2
		}
	}
	bookID, err := resolveBookID(*serverURL, flags.Arg(0))
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	if cover.value != nil {
		data, err := loadCover(*cover.value)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "load cover: %v\n", err)
			return 1
		}
		changes["cover"] = data
	}
	if len(authorsFlag) > 0 {
		ids, err := resolveExistingAuthors(*serverURL, authorsFlag)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 1
		}
		changes["author_ids"] = ids
	}
	endpoint, err := joinURL(*serverURL, "/api/books/"+strconv.FormatInt(bookID, 10))
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "invalid server URL: %v\n", err)
		return 2
	}
	var updated books.Book
	if err := patchEndpoint(endpoint, changes, &updated); err != nil {
		_, _ = fmt.Fprintf(stderr, "edit book: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "%d\t%s\n", updated.ID, updated.Title)
	return 0
}

func runBooksLS(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("books ls", flag.ContinueOnError)
	flags.SetOutput(stderr)

	serverURL := flags.String("server", defaultServerURL, "Bookist server URL")
	formatValue := flags.String("format", string(outputFormatPretty), "Output format (tsv|pretty|json)")
	query := flags.String("query", "", "Filter books by title")
	listRef := flags.String("list", "", "Filter books by list name or ID")

	help := commandHelp{
		name:        "bookist books ls",
		usage:       "bookist books ls [options]",
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

	var listedBooks []books.Book

	if strings.TrimSpace(*listRef) == "" {
		listedBooks, err = fetchBooks(*serverURL, *query)
	} else {
		var listID int64

		listID, err = resolveListID(*serverURL, *listRef)
		if err == nil {
			listedBooks, err = fetchBooksByListID(*serverURL, listID, *query)
		}
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "ls books: %v\n", err)
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
		_, _ = fmt.Fprintf(stderr, "ls books: write output: %v\n", err)
		return 1
	}

	return 0
}

func runBooksAdd(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("books add", flag.ContinueOnError)

	serverURL := flags.String("server", defaultServerURL, "Bookist server URL")
	title := flags.String("title", "", "Book title")
	coverSource := flags.String("cover", "", "Cover image file path or URL")

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
		authorIDs, err := resolveOrCreateAuthors(*serverURL, authorFlags)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "%v\n", err)
			return 1
		}

		input.AuthorIDs = authorIDs
	}

	if *coverSource != "" {
		cover, err := loadCover(*coverSource)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "load cover: %v\n", err)
			return 1
		}

		input.Cover = &cover
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

func runBooksRM(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("books rm", flag.ContinueOnError)

	serverURL := flags.String("server", defaultServerURL, "Bookist server URL")

	help := commandHelp{
		name:        "bookist books rm",
		usage:       "bookist books rm [options] <title-or-ID>",
		description: "Remove a book",
	}

	if ok, exitCode := parseFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
	}

	if flags.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "Error: books rm requires exactly one title or ID")
		_, _ = fmt.Fprintln(stderr)

		printCommandHelp(stderr, help, flags)

		return 2
	}

	bookID, err := resolveBookID(*serverURL, flags.Arg(0))
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	endpoint, err := joinURL(*serverURL, "/api/books/"+strconv.FormatInt(bookID, 10))
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "invalid server URL: %v\n", err)
		return 2
	}

	if err := deleteEndpoint(endpoint); err != nil {
		_, _ = fmt.Fprintf(stderr, "remove book: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "removed book %d\n", bookID)

	return 0
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

func fetchBooks(serverURL, query string) ([]books.Book, error) {
	endpoint, err := joinURLWithQuery(serverURL, "/api/books", query)
	if err != nil {
		return nil, fmt.Errorf("invalid server URL: %v", err)
	}

	return fetchBooksFromEndpoint(endpoint)
}

func fetchBooksByListID(serverURL string, listID int64, query string) ([]books.Book, error) {
	path := "/api/lists/" + strconv.FormatInt(listID, 10) + "/books"
	endpoint, err := joinURLWithQuery(serverURL, path, query)
	if err != nil {
		return nil, fmt.Errorf("invalid server URL: %v", err)
	}

	return fetchBooksFromEndpoint(endpoint)
}

func fetchBooksFromEndpoint(endpoint string) ([]books.Book, error) {
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

func loadCover(source string) ([]byte, error) {
	parsed, err := url.Parse(source)
	if err != nil {
		return nil, err
	}

	var reader io.ReadCloser
	switch parsed.Scheme {
	case "":
		reader, err = os.Open(source)
		if err != nil {
			return nil, err
		}
	case "http", "https":
		client := http.Client{
			Timeout: 10 * time.Second,
			CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
				return http.ErrUseLastResponse
			},
		}

		var response *http.Response
		response, err = client.Get(source)
		if err != nil {
			return nil, err
		}

		if response.StatusCode < http.StatusOK || response.StatusCode >= http.StatusMultipleChoices {
			_ = response.Body.Close()
			return nil, fmt.Errorf("cover URL returned %s", response.Status)
		}

		reader = response.Body
	default:
		return nil, fmt.Errorf("unsupported URL scheme %q", parsed.Scheme)
	}

	defer func() {
		_ = reader.Close()
	}()

	data, err := io.ReadAll(io.LimitReader(reader, covers.MaxSize+1))
	if err != nil {
		return nil, err
	}

	if err := covers.Validate(data); err != nil {
		return nil, err
	}

	return data, nil
}
