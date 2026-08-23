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

	"bakku.dev/bookist/internal/reads"
)

func runReads(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "Error: missing reads command")
		_, _ = fmt.Fprintln(stderr)
		printReadsHelp(stderr)
		return 2
	}

	switch args[0] {
	case "ls":
		return runReadsLS(args[1:], stdout, stderr)
	case "add":
		return runReadsAdd(args[1:], stdout, stderr)
	case "rm":
		return runReadsRM(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		printReadsHelp(stdout)
		return 0
	default:
		_, _ = fmt.Fprintf(stderr, "Error: unknown reads command %q\n\n", args[0])
		printReadsHelp(stderr)
		return 2
	}
}

func printReadsHelp(w io.Writer) {
	printCommandHelp(w, commandHelp{
		name:        "bookist reads",
		usage:       "bookist reads [command [command options]]",
		description: "Manage book reads",
		commands: []helpCommand{
			{name: "ls", description: "List reads for a book"},
			{name: "add", description: "Record a read for a book"},
			{name: "rm", description: "Remove a read"},
		},
	}, nil)
}

func runReadsLS(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("reads ls", flag.ContinueOnError)

	serverURL := flags.String("server", defaultServerURL, "Bookist server URL")
	bookRef := flags.String("book", "", "Book title or ID")
	formatValue := flags.String("format", string(outputFormatPretty), "Output format (tsv|pretty|json)")

	help := commandHelp{
		name:        "bookist reads ls",
		usage:       "bookist reads ls [options]",
		description: "List reads for a book",
	}

	if ok, exitCode := parseFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
	}

	format, err := parseOutputFormat(*formatValue)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}

	if strings.TrimSpace(*bookRef) == "" {
		_, _ = fmt.Fprintln(stderr, "--book is required")
		return 2
	}

	bookID, err := resolveBookID(*serverURL, *bookRef)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	listed, err := fetchReads(*serverURL, bookID)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "list reads: %v\n", err)
		return 1
	}

	rows := make([][]string, 0, len(listed))

	for _, read := range listed {
		rows = append(rows, []string{
			strconv.FormatInt(read.ID, 10),
			stringValue(read.StartedAt),
			stringValue(read.FinishedAt),
			stringValue(read.AbandonedAt),
			floatValue(read.Rating),
			stringValue(read.Notes),
		})
	}

	if err := writeListOutput(stdout, format, listed,
		[]string{"ID", "STARTED_AT", "FINISHED_AT", "ABANDONED_AT", "RATING", "NOTES"}, rows); err != nil {
		_, _ = fmt.Fprintf(stderr, "list reads: write output: %v\n", err)
		return 1
	}

	return 0
}

func runReadsAdd(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("reads add", flag.ContinueOnError)

	serverURL := flags.String("server", defaultServerURL, "Bookist server URL")
	bookRef := flags.String("book", "", "Book title or ID")

	var startedAt optionalStringFlag
	var finishedAt optionalStringFlag
	var abandonedAt optionalStringFlag
	var rating optionalFloatFlag
	var notes optionalStringFlag

	flags.Var(&startedAt, "started-at", "Date reading started (YYYY-MM-DD)")
	flags.Var(&finishedAt, "finished-at", "Date reading finished (YYYY-MM-DD)")
	flags.Var(&abandonedAt, "abandoned-at", "Date reading abandoned (YYYY-MM-DD)")
	flags.Var(&rating, "rating", "Rating from 1 to 5 in increments of 0.5")
	flags.Var(&notes, "notes", "Read notes")

	help := commandHelp{
		name:        "bookist reads add",
		usage:       "bookist reads add [options]",
		description: "Record a read for a book",
	}

	if ok, exitCode := parseFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
	}

	if strings.TrimSpace(*bookRef) == "" {
		_, _ = fmt.Fprintln(stderr, "--book is required")
		return 2
	}

	bookID, err := resolveBookID(*serverURL, *bookRef)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	input := reads.CreateReadRequest{
		StartedAt:   startedAt.value,
		FinishedAt:  finishedAt.value,
		AbandonedAt: abandonedAt.value,
		Rating:      rating.value,
		Notes:       notes.value,
	}

	body, err := json.Marshal(input)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "encode read: %v\n", err)
		return 1
	}

	endpoint, err := joinURL(*serverURL, "/api/books/"+strconv.FormatInt(bookID, 10)+"/reads")
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "invalid server URL: %v\n", err)
		return 2
	}

	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Post(endpoint, "application/json", bytes.NewReader(body))
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "add read: %v\n", err)
		return 1
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusCreated {
		_, _ = fmt.Fprintf(stderr, "add read: server returned %s\n", resp.Status)
		return 1
	}

	var created reads.Read
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		_, _ = fmt.Fprintf(stderr, "decode read: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "%d\t%d\n", created.ID, created.BookID)

	return 0
}

func runReadsRM(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("reads rm", flag.ContinueOnError)

	serverURL := flags.String("server", defaultServerURL, "Bookist server URL")

	help := commandHelp{
		name:        "bookist reads rm",
		usage:       "bookist reads rm [options] <read-ID>",
		description: "Remove a read",
	}

	if ok, exitCode := parseFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
	}

	if flags.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "Error: reads rm requires exactly one read ID")
		_, _ = fmt.Fprintln(stderr)
		printCommandHelp(stderr, help, flags)
		return 2
	}

	readRef := strings.TrimSpace(flags.Arg(0))

	readID, isID, err := parseIDReference(readRef)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%v\n", err)
		return 2
	}

	if !isID {
		_, _ = fmt.Fprintf(stderr, "invalid read ID %q\n", readRef)
		return 2
	}

	endpoint, err := joinURL(*serverURL, "/api/reads/"+strconv.FormatInt(readID, 10))
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "invalid server URL: %v\n", err)
		return 2
	}

	if err := deleteEndpoint(endpoint); err != nil {
		_, _ = fmt.Fprintf(stderr, "remove read: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "removed read %d\n", readID)

	return 0
}

func fetchReads(serverURL string, bookID int64) ([]reads.Read, error) {
	endpoint, err := joinURL(serverURL, "/api/books/"+strconv.FormatInt(bookID, 10)+"/reads")
	if err != nil {
		return nil, fmt.Errorf("invalid server URL: %v", err)
	}

	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Get(endpoint)
	if err != nil {
		return nil, fmt.Errorf("fetch reads: %v", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch reads: server returned %s", resp.Status)
	}

	var listed []reads.Read
	if err := json.NewDecoder(resp.Body).Decode(&listed); err != nil {
		return nil, fmt.Errorf("decode reads: %v", err)
	}

	return listed, nil
}
