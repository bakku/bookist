package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"bakku.dev/bookist/internal/reads"
)

func readClearTargets(request *reads.UpdateReadRequest) map[string]clearTarget {
	return map[string]clearTarget{
		"started-at":   &request.StartedAt,
		"finished-at":  &request.FinishedAt,
		"abandoned-at": &request.AbandonedAt,
		"rating":       &request.Rating,
		"notes":        &request.Notes,
	}
}

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
	case "show":
		return runReadsShow(args[1:], stdout, stderr)
	case "add":
		return runReadsAdd(args[1:], stdout, stderr)
	case "edit":
		return runReadsEdit(args[1:], stdout, stderr)
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
			{name: "show", description: "Show a read"},
			{name: "add", description: "Record a read for a book"},
			{name: "edit", description: "Edit a read"},
			{name: "rm", description: "Remove a read"},
		},
	}, nil)
}

func runReadsEdit(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("reads edit", flag.ContinueOnError)

	serverURL := flags.String("server", defaultServerURL, "Bookist server URL")

	var startedAt optionalStringFlag
	var finishedAt optionalStringFlag
	var abandonedAt optionalStringFlag
	var notes optionalStringFlag
	var rating optionalFloatFlag
	var clears stringSliceFlag

	flags.Var(&startedAt, "started-at", "Date reading started (YYYY-MM-DD)")
	flags.Var(&finishedAt, "finished-at", "Date reading finished (YYYY-MM-DD)")
	flags.Var(&abandonedAt, "abandoned-at", "Date reading abandoned (YYYY-MM-DD)")
	flags.Var(&rating, "rating", "Rating from 1 to 5 in increments of 0.5")
	flags.Var(&notes, "notes", "Read notes")
	flags.Var(&clears, "clear", clearFlagUsage(readClearTargets(&reads.UpdateReadRequest{})))

	help := commandHelp{
		name:        "bookist reads edit",
		usage:       "bookist reads edit <read-ID> [options]",
		description: "Edit a read",
		details:     []string{partialUpdateHelp},
		examples: []string{
			"bookist reads edit 9 --rating 4.5",
			"bookist reads edit 9 --clear rating --clear notes",
			"bookist reads edit 9 --clear abandoned-at --finished-at 2026-09-16",
		},
	}

	if ok, exitCode := parseEditFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
	}

	if flags.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "Error: reads edit requires exactly one read ID")
		_, _ = fmt.Fprintln(stderr)
		printCommandHelp(stderr, help, flags)
		return 2
	}

	request := reads.UpdateReadRequest{
		StartedAt:   supplied(startedAt.value),
		FinishedAt:  supplied(finishedAt.value),
		AbandonedAt: supplied(abandonedAt.value),
		Rating:      supplied(rating.value),
		Notes:       supplied(notes.value),
	}
	if err := applyClears(readClearTargets(&request), clears); err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}
	if isEmptyUpdate(request) {
		_, _ = fmt.Fprintln(stderr, "Error: reads edit requires at least one change")
		return 2
	}

	readRef := strings.TrimSpace(flags.Arg(0))
	readID, isID, err := parseIDReference(readRef)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	if !isID {
		_, _ = fmt.Fprintf(stderr, "invalid read ID %q\n", readRef)
		return 2
	}

	c := newClient(*serverURL, stderr)
	if c == nil {
		return 2
	}

	updated, err := c.Reads.Update(context.Background(), readID, request)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "edit read: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "%d\t%d\n", updated.ID, updated.BookID)
	return 0
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

	ctx := context.Background()
	c := newClient(*serverURL, stderr)
	if c == nil {
		return 2
	}

	bookID, err := resolveBookID(ctx, c, *bookRef)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	listed, err := c.Reads.List(ctx, bookID)
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

func runReadsShow(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("reads show", flag.ContinueOnError)

	serverURL := flags.String("server", defaultServerURL, "Bookist server URL")
	formatValue := flags.String("format", string(outputFormatPretty), "Output format (tsv|pretty|json)")

	help := commandHelp{
		name:        "bookist reads show",
		usage:       "bookist reads show <read-ID> [options]",
		description: "Show a read",
		examples:    []string{"bookist reads show 9", "bookist reads show 9 --format json"},
	}
	if ok, exitCode := parseEditFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
	}

	if flags.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "Error: reads show requires exactly one read ID")
		_, _ = fmt.Fprintln(stderr)
		printCommandHelp(stderr, help, flags)
		return 2
	}

	format, err := parseOutputFormat(*formatValue)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}

	readRef := strings.TrimSpace(flags.Arg(0))
	readID, isID, err := parseIDReference(readRef)
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 2
	}
	if !isID {
		_, _ = fmt.Fprintf(stderr, "invalid read ID %q\n", readRef)
		return 2
	}

	c := newClient(*serverURL, stderr)
	if c == nil {
		return 2
	}

	read, err := c.Reads.Get(context.Background(), readID)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "show read: %v\n", err)
		return 1
	}

	fields := []showField{
		{"ID", strconv.FormatInt(read.ID, 10)},
		{"Book ID", strconv.FormatInt(read.BookID, 10)},
		{"Started at", stringValue(read.StartedAt)},
		{"Finished at", stringValue(read.FinishedAt)},
		{"Abandoned at", stringValue(read.AbandonedAt)},
		{"Rating", floatValue(read.Rating)},
		{"Notes", stringValue(read.Notes)},
		{"Created at", timeValue(read.CreatedAt)},
		{"Updated at", timeValue(read.UpdatedAt)},
	}
	if err := writeShowOutput(stdout, format, read, fields); err != nil {
		_, _ = fmt.Fprintf(stderr, "show read: write output: %v\n", err)
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

	ctx := context.Background()
	c := newClient(*serverURL, stderr)
	if c == nil {
		return 2
	}

	bookID, err := resolveBookID(ctx, c, *bookRef)
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

	created, err := c.Reads.Create(ctx, bookID, input)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "add read: %v\n", err)
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

	c := newClient(*serverURL, stderr)
	if c == nil {
		return 2
	}

	if err := c.Reads.Delete(context.Background(), readID); err != nil {
		_, _ = fmt.Fprintf(stderr, "remove read: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "removed read %d\n", readID)

	return 0
}
