package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"strconv"
	"strings"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/client"
	"bakku.dev/bookist/internal/covers"
	"bakku.dev/bookist/internal/optional"
)

func bookClearTargets(request *books.UpdateBookRequest) map[string]clearTarget {
	return map[string]clearTarget{
		"isbn":               &request.ISBN,
		"authors":            &request.AuthorIDs,
		"language":           &request.Language,
		"publisher":          &request.Publisher,
		"edition":            &request.Edition,
		"format":             &request.Format,
		"purchased-at":       &request.PurchasedAt,
		"purchase-price":     &request.PurchasePrice,
		"pages":              &request.Pages,
		"notes":              &request.Notes,
		"summary":            &request.Summary,
		"series-name":        &request.SeriesName,
		"series-position":    &request.SeriesPosition,
		"location":           &request.Location,
		"condition":          &request.Condition,
		"acquisition-source": &request.AcquisitionSource,
		"published-year":     &request.PublishedYear,
		"published-month":    &request.PublishedMonth,
		"published-day":      &request.PublishedDay,
		"cover":              &request.Cover,
	}
}

func runBooks(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "Error: missing books command")
		_, _ = fmt.Fprintln(stderr)
		printBooksHelp(stderr)
		return 2
	}

	switch args[0] {
	case "ls", "list":
		return runBooksLS(args[1:], stdout, stderr)

	case "show":
		return runBooksShow(args[1:], stdout, stderr)

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
			{name: "ls, list", description: "List books"},
			{name: "show", description: "Show a book"},
			{name: "add", description: "Add a book"},
			{name: "edit", description: "Edit a book"},
			{name: "rm", description: "Remove a book"},
		},
	}, nil)
}

func runBooksEdit(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("books edit", flag.ContinueOnError)
	serverURL := serverFlag(flags)
	outputValue := outputFlag(flags, outputFormatText, mutationFormatsHelp)
	var title, cover optionalStringFlag
	var authorsFlag, clears stringSliceFlag
	var isbn, language, publisher, edition, format, purchasedAt, purchasePrice optionalStringFlag
	var notes, summary, seriesName, location, condition, acquisitionSource optionalStringFlag
	var seriesPosition optionalFloatFlag
	var pages, publishedYear, publishedMonth, publishedDay optionalIntFlag
	flags.Var(&title, "title", "Book title")
	flags.Var(&authorsFlag, "author", "Existing author name, ID or id:N; supplied authors replace the complete author list (repeatable)")
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
	flags.Var(&clears, "clear", clearFlagUsage(bookClearTargets(&books.UpdateBookRequest{})))
	help := commandHelp{
		name:        "bookist books edit",
		usage:       "bookist books edit <title-or-ID> [options]",
		description: "Edit a book",
		details: []string{
			partialUpdateHelp,
			"Supplying --author replaces the complete author list with existing authors; omitting it preserves current authors, and --clear authors removes all author relationships.",
			referenceHelp,
			mutationOutputHelp,
		},
		examples: []string{
			`bookist books edit 12 --title "Dune Messiah"`,
			"bookist books edit 12 --clear isbn --clear notes",
			"bookist books edit 12 --author Alice --author Bob  # complete author list becomes Alice and Bob",
		},
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

	request := books.UpdateBookRequest{
		Title:             supplied(title.value),
		ISBN:              supplied(isbn.value),
		Language:          supplied(language.value),
		Publisher:         supplied(publisher.value),
		Edition:           supplied(edition.value),
		PurchasedAt:       supplied(purchasedAt.value),
		PurchasePrice:     supplied(purchasePrice.value),
		Pages:             supplied(pages.value),
		Notes:             supplied(notes.value),
		Summary:           supplied(summary.value),
		SeriesName:        supplied(seriesName.value),
		SeriesPosition:    supplied(seriesPosition.value),
		Location:          supplied(location.value),
		AcquisitionSource: supplied(acquisitionSource.value),
		PublishedYear:     supplied(publishedYear.value),
		PublishedMonth:    supplied(publishedMonth.value),
		PublishedDay:      supplied(publishedDay.value),
	}
	if format.value != nil {
		request.Format = optional.Set(books.Format(*format.value))
	}
	if condition.value != nil {
		request.Condition = optional.Set(books.Condition(*condition.value))
	}
	// Authors and the cover are resolved after validation; until then a placeholder
	// marks them as supplied so that --clear conflicts and empty edits are detected.
	if len(authorsFlag) > 0 {
		request.AuthorIDs = optional.Set([]int64{})
	}
	if cover.value != nil {
		request.Cover = optional.Set([]byte{})
	}
	if err := applyClears(bookClearTargets(&request), clears); err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}
	if isEmptyUpdate(request) {
		_, _ = fmt.Fprintln(stderr, "Error: books edit requires at least one change")
		return 2
	}
	for _, author := range authorsFlag {
		if strings.TrimSpace(author) == "" {
			_, _ = fmt.Fprintln(stderr, "Error: --author must not be blank; use --clear authors")
			return 2
		}
	}

	printFormat, err := parseMutationOutput(*outputValue)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}

	ctx := context.Background()
	c := newClient(*serverURL, stderr)
	if c == nil {
		return 2
	}

	bookID, err := resolveBookID(ctx, c, flags.Arg(0))
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
		request.Cover = optional.Set(data)
	}
	if len(authorsFlag) > 0 {
		ids, err := resolveAuthors(ctx, c, authorsFlag, false, stderr)
		if err != nil {
			_, _ = fmt.Fprintln(stderr, err)
			return 1
		}
		request.AuthorIDs = optional.Set(ids)
	}

	updated, err := c.Books.Update(ctx, bookID, request)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "edit book: %v\n", err)
		return 1
	}
	text := fmt.Sprintf("%d\t%s", updated.ID, updated.Title)
	if err := writeMutationOutput(stdout, printFormat, updated, text); err != nil {
		_, _ = fmt.Fprintf(stderr, "edit book: write output: %v\n", err)
		return 1
	}

	return 0
}

func runBooksLS(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("books ls", flag.ContinueOnError)
	flags.SetOutput(stderr)

	serverURL := serverFlag(flags)
	outputValue := outputFlag(flags, outputFormatPretty, outputFormatsHelp)
	query := flags.String("query", "", "Filter books by title")
	listRef := flags.String("list", "", "Filter books by list name, ID or id:N")

	help := commandHelp{
		name:        "bookist books ls",
		usage:       "bookist books ls [options]",
		description: "List books",
	}
	if ok, exitCode := parseFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
	}
	if !rejectArgs(flags, "books ls", stderr, help) {
		return 2
	}

	format, err := parseOutputFormat(*outputValue)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}

	ctx := context.Background()
	c := newClient(*serverURL, stderr)
	if c == nil {
		return 2
	}

	var listedBooks []books.Book

	if strings.TrimSpace(*listRef) == "" {
		listedBooks, err = c.Books.List(ctx, *query)
	} else {
		var listID int64

		listID, err = resolveListID(ctx, c, *listRef)
		if err == nil {
			listedBooks, err = c.Lists.ListBooks(ctx, listID, *query)
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

		rows = append(rows, []string{strconv.FormatInt(book.ID, 10), book.Title, authorNames(book.Authors), isbn})
	}

	if err := writeListOutput(stdout, format, listedBooks, []string{"ID", "TITLE", "AUTHORS", "ISBN"}, rows); err != nil {
		_, _ = fmt.Fprintf(stderr, "ls books: write output: %v\n", err)
		return 1
	}

	return 0
}

func runBooksShow(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("books show", flag.ContinueOnError)

	serverURL := serverFlag(flags)
	outputValue := outputFlag(flags, outputFormatPretty, outputFormatsHelp)

	help := commandHelp{
		name:        "bookist books show",
		usage:       "bookist books show <title-or-ID> [options]",
		description: "Show a book",
		details:     []string{referenceHelp},
		examples: []string{
			"bookist books show 12",
			"bookist books show id:12",
			`bookist books show Dune --output json`,
		},
	}
	if ok, exitCode := parseFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
	}

	if flags.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "Error: books show requires exactly one title or ID")
		_, _ = fmt.Fprintln(stderr)
		printCommandHelp(stderr, help, flags)
		return 2
	}

	format, err := parseOutputFormat(*outputValue)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}

	ctx := context.Background()
	c := newClient(*serverURL, stderr)
	if c == nil {
		return 2
	}

	bookID, err := resolveBookID(ctx, c, flags.Arg(0))
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	book, err := c.Books.Get(ctx, bookID)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "show book: %v\n", err)
		return 1
	}

	fields := []showField{
		{"ID", strconv.FormatInt(book.ID, 10)},
		{"Title", book.Title},
		{"ISBN", stringValue(book.ISBN)},
		{"Authors", authorNames(book.Authors)},
		{"Language", stringValue(book.Language)},
		{"Publisher", stringValue(book.Publisher)},
		{"Edition", stringValue(book.Edition)},
		{"Format", bookFormatValue(book.Format)},
		{"Purchased at", stringValue(book.PurchasedAt)},
		{"Purchase price", stringValue(book.PurchasePrice)},
		{"Pages", intValue(book.Pages)},
		{"Notes", stringValue(book.Notes)},
		{"Summary", stringValue(book.Summary)},
		{"Series name", stringValue(book.SeriesName)},
		{"Series position", floatValue(book.SeriesPosition)},
		{"Location", stringValue(book.Location)},
		{"Condition", bookConditionValue(book.Condition)},
		{"Acquisition source", stringValue(book.AcquisitionSource)},
		{"Published", publishedValue(book)},
		{"Cover URL", stringValue(book.CoverURL)},
		{"Created at", timeValue(book.CreatedAt)},
		{"Updated at", timeValue(book.UpdatedAt)},
	}
	if err := writeShowOutput(stdout, format, book, fields); err != nil {
		_, _ = fmt.Fprintf(stderr, "show book: write output: %v\n", err)
		return 1
	}

	return 0
}

// authorNames joins author names with commas.
func authorNames(bookAuthors []authors.Author) string {
	names := make([]string, 0, len(bookAuthors))
	for _, author := range bookAuthors {
		names = append(names, author.Name)
	}

	return strings.Join(names, ", ")
}

// publishedValue formats the publication date as far as it is known: YYYY,
// YYYY-MM or YYYY-MM-DD.
func publishedValue(book books.Book) string {
	if book.PublishedYear == nil {
		return ""
	}

	published := fmt.Sprintf("%04d", *book.PublishedYear)
	if book.PublishedMonth == nil {
		return published
	}

	published += fmt.Sprintf("-%02d", *book.PublishedMonth)
	if book.PublishedDay == nil {
		return published
	}

	return published + fmt.Sprintf("-%02d", *book.PublishedDay)
}

func bookFormatValue(value *books.Format) string {
	if value == nil {
		return ""
	}
	return string(*value)
}

func bookConditionValue(value *books.Condition) string {
	if value == nil {
		return ""
	}
	return string(*value)
}

func runBooksAdd(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("books add", flag.ContinueOnError)

	serverURL := serverFlag(flags)
	outputValue := outputFlag(flags, outputFormatText, mutationFormatsHelp)
	title := flags.String("title", "", "Book title")
	coverSource := flags.String("cover", "", "Cover image file path or URL")

	var authorFlags stringSliceFlag
	flags.Var(&authorFlags, "author", "Existing author name, ID or id:N (repeatable)")
	createAuthors := flags.Bool("create-authors", false, "Create --author values that match no existing author, reporting each on stderr")

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
		details: []string{
			"Each --author must be an existing author unless --create-authors is given, which creates missing authors and reports each on stderr.",
			referenceHelp,
			mutationOutputHelp,
		},
		examples: []string{
			`bookist books add --title Dune --author "Frank Herbert" --create-authors`,
		},
	}
	if ok, exitCode := parseFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
	}
	if !rejectArgs(flags, "books add", stderr, help) {
		return 2
	}

	for _, author := range authorFlags {
		if strings.TrimSpace(author) == "" {
			_, _ = fmt.Fprintln(stderr, "Error: --author must not be blank")
			return 2
		}
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

	printFormat, err := parseMutationOutput(*outputValue)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}

	ctx := context.Background()
	c := newClient(*serverURL, stderr)
	if c == nil {
		return 2
	}

	// The cover is loaded first so that a bad cover cannot leave newly created
	// authors behind.
	if *coverSource != "" {
		cover, err := loadCover(*coverSource)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "load cover: %v\n", err)
			return 1
		}

		input.Cover = &cover
	}

	if len(authorFlags) > 0 {
		authorIDs, err := resolveAuthors(ctx, c, authorFlags, *createAuthors, stderr)
		if err != nil {
			_, _ = fmt.Fprintf(stderr, "%v\n", err)
			return 1
		}

		input.AuthorIDs = authorIDs
	}

	book, err := c.Books.Create(ctx, input)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "add book: %v\n", err)
		return 1
	}

	text := fmt.Sprintf("%d\t%s", book.ID, book.Title)
	if err := writeMutationOutput(stdout, printFormat, book, text); err != nil {
		_, _ = fmt.Fprintf(stderr, "add book: write output: %v\n", err)
		return 1
	}

	return 0
}

func runBooksRM(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("books rm", flag.ContinueOnError)

	serverURL := serverFlag(flags)
	outputValue := outputFlag(flags, outputFormatText, mutationFormatsHelp)

	help := commandHelp{
		name:        "bookist books rm",
		usage:       "bookist books rm [options] <title-or-ID>",
		description: "Remove a book",
		details:     []string{referenceHelp, removedOutputHelp},
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

	format, err := parseMutationOutput(*outputValue)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}

	ctx := context.Background()
	c := newClient(*serverURL, stderr)
	if c == nil {
		return 2
	}

	bookID, err := resolveBookID(ctx, c, flags.Arg(0))
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	if err := c.Books.Delete(ctx, bookID); err != nil {
		_, _ = fmt.Fprintf(stderr, "remove book: %v\n", err)
		return 1
	}

	text := fmt.Sprintf("removed book %d", bookID)
	if err := writeMutationOutput(stdout, format, removedResult{ID: bookID, Removed: true}, text); err != nil {
		_, _ = fmt.Fprintf(stderr, "remove book: write output: %v\n", err)
		return 1
	}

	return 0
}

func resolveBookID(ctx context.Context, c *client.Client, value string) (int64, error) {
	return resolveReference(ctx, referenceSource{
		kind: "book",
		get: func(ctx context.Context, id int64) (record, error) {
			book, err := c.Books.Get(ctx, id)
			return record{ID: book.ID, Name: book.Title}, err
		},
		find: func(ctx context.Context, title string) ([]record, error) {
			existing, err := c.Books.List(ctx, title)
			if err != nil {
				return nil, err
			}

			var matches []record
			for _, book := range existing {
				if strings.EqualFold(book.Title, title) {
					matches = append(matches, record{ID: book.ID, Name: book.Title})
				}
			}

			return matches, nil
		},
	}, value)
}

// coverTransport is the transport used to download covers; nil uses the
// default transport.
var coverTransport http.RoundTripper

// maxCoverRedirects is Go's default redirect limit.
const maxCoverRedirects = 10

// checkCoverRedirect follows redirects up to Go's default limit but only to
// http and https URLs.
func checkCoverRedirect(req *http.Request, via []*http.Request) error {
	if len(via) >= maxCoverRedirects {
		return fmt.Errorf("stopped after %d redirects", maxCoverRedirects)
	}
	if req.URL.Scheme != "http" && req.URL.Scheme != "https" {
		return fmt.Errorf("unsupported redirect URL scheme %q", req.URL.Scheme)
	}

	return nil
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
		// The deadline covers the whole download, including redirects and
		// reading the body.
		ctx, cancel := context.WithTimeout(context.Background(), client.CoverTimeout)
		defer cancel()

		request, err := http.NewRequestWithContext(ctx, http.MethodGet, source, nil)
		if err != nil {
			return nil, err
		}

		httpClient := http.Client{
			Transport:     coverTransport,
			CheckRedirect: checkCoverRedirect,
		}

		var response *http.Response
		response, err = httpClient.Do(request)
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
