package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"bakku.dev/bookist/internal/client"
	"bakku.dev/bookist/internal/lists"
)

func listClearTargets(request *lists.UpdateListRequest) map[string]clearTarget {
	return map[string]clearTarget{"description": &request.Description}
}

func runLists(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "Error: missing lists command")
		_, _ = fmt.Fprintln(stderr)
		printListsHelp(stderr)
		return 2
	}

	switch args[0] {
	case "ls", "list":
		return runListsLS(args[1:], stdout, stderr)

	case "show":
		return runListsShow(args[1:], stdout, stderr)

	case "add":
		return runListsAdd(args[1:], stdout, stderr)

	case "edit":
		return runListsEdit(args[1:], stdout, stderr)

	case "add-book":
		return runListsAddBook(args[1:], stdout, stderr)

	case "rm":
		return runListsRM(args[1:], stdout, stderr)

	case "rm-book":
		return runListsRMBook(args[1:], stdout, stderr)

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
			{name: "ls, list", description: "List book lists"},
			{name: "show", description: "Show a book list"},
			{name: "add", description: "Add a book list"},
			{name: "edit", description: "Edit a book list"},
			{name: "add-book", description: "Add a book to a list"},
			{name: "rm", description: "Remove a book list"},
			{name: "rm-book", description: "Remove a book from a list"},
		},
	}, nil)
}

func runListsEdit(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("lists edit", flag.ContinueOnError)

	serverURL := serverFlag(flags)
	outputValue := outputFlag(flags, outputFormatText, mutationFormatsHelp)

	var name optionalStringFlag
	var description optionalStringFlag
	var clears stringSliceFlag

	flags.Var(&name, "name", "List name")
	flags.Var(&description, "description", "List description")
	flags.Var(&clears, "clear", clearFlagUsage(listClearTargets(&lists.UpdateListRequest{})))

	help := commandHelp{
		name:        "bookist lists edit",
		usage:       "bookist lists edit <name-or-ID> [options]",
		description: "Edit a book list",
		details:     []string{partialUpdateHelp, referenceHelp, mutationOutputHelp},
		examples: []string{
			`bookist lists edit 5 --name "Best Books"`,
			"bookist lists edit 5 --clear description",
		},
	}

	if ok, exitCode := parseFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
	}

	if flags.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "Error: lists edit requires exactly one name or ID")
		_, _ = fmt.Fprintln(stderr)
		printCommandHelp(stderr, help, flags)
		return 2
	}

	request := lists.UpdateListRequest{
		Name:        supplied(name.value),
		Description: supplied(description.value),
	}
	if err := applyClears(listClearTargets(&request), clears); err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}
	if isEmptyUpdate(request) {
		_, _ = fmt.Fprintln(stderr, "Error: lists edit requires at least one change")
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

	listID, err := resolveListID(ctx, c, flags.Arg(0))
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	updated, err := c.Lists.Update(ctx, listID, request)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "edit list: %v\n", err)
		return 1
	}

	text := fmt.Sprintf("%d\t%s", updated.ID, updated.Name)
	if err := writeMutationOutput(stdout, format, updated, text); err != nil {
		_, _ = fmt.Fprintf(stderr, "edit list: write output: %v\n", err)
		return 1
	}

	return 0
}

func runListsLS(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("lists ls", flag.ContinueOnError)

	serverURL := serverFlag(flags)
	outputValue := outputFlag(flags, outputFormatPretty, outputFormatsHelp)
	query := flags.String("query", "", "Filter lists by name")

	help := commandHelp{
		name:        "bookist lists ls",
		usage:       "bookist lists ls [options]",
		description: "List book lists",
	}
	if ok, exitCode := parseFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
	}
	if !rejectArgs(flags, "lists ls", stderr, help) {
		return 2
	}

	format, err := parseOutputFormat(*outputValue)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}

	c := newClient(*serverURL, stderr)
	if c == nil {
		return 2
	}

	listed, err := c.Lists.List(context.Background(), *query)
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

func runListsShow(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("lists show", flag.ContinueOnError)

	serverURL := serverFlag(flags)
	outputValue := outputFlag(flags, outputFormatPretty, outputFormatsHelp)

	help := commandHelp{
		name:        "bookist lists show",
		usage:       "bookist lists show <name-or-ID> [options]",
		description: "Show a book list",
		details:     []string{referenceHelp},
		examples: []string{
			"bookist lists show 5",
			"bookist lists show id:5",
			`bookist lists show "Best Books" --output json`,
		},
	}
	if ok, exitCode := parseFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
	}

	if flags.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "Error: lists show requires exactly one name or ID")
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

	listID, err := resolveListID(ctx, c, flags.Arg(0))
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	list, err := c.Lists.Get(ctx, listID)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "show list: %v\n", err)
		return 1
	}

	fields := []showField{
		{"ID", strconv.FormatInt(list.ID, 10)},
		{"Name", list.Name},
		{"Description", stringValue(list.Description)},
		{"Created at", timeValue(list.CreatedAt)},
		{"Updated at", timeValue(list.UpdatedAt)},
	}
	if err := writeShowOutput(stdout, format, list, fields); err != nil {
		_, _ = fmt.Fprintf(stderr, "show list: write output: %v\n", err)
		return 1
	}

	return 0
}

func runListsAdd(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("lists add", flag.ContinueOnError)

	serverURL := serverFlag(flags)
	outputValue := outputFlag(flags, outputFormatText, mutationFormatsHelp)
	name := flags.String("name", "", "List name")
	description := flags.String("description", "", "List description")

	help := commandHelp{
		name:        "bookist lists add",
		usage:       "bookist lists add [options]",
		description: "Add a book list",
		details:     []string{mutationOutputHelp},
	}
	if ok, exitCode := parseFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
	}
	if !rejectArgs(flags, "lists add", stderr, help) {
		return 2
	}

	trimmedName := strings.TrimSpace(*name)

	input := lists.CreateListRequest{Name: trimmedName}

	if strings.TrimSpace(*description) != "" {
		input.Description = description
	}

	format, err := parseMutationOutput(*outputValue)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}

	c := newClient(*serverURL, stderr)
	if c == nil {
		return 2
	}

	created, err := c.Lists.Create(context.Background(), input)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "add list: %v\n", err)
		return 1
	}

	text := fmt.Sprintf("%d\t%s", created.ID, created.Name)
	if err := writeMutationOutput(stdout, format, created, text); err != nil {
		_, _ = fmt.Fprintf(stderr, "add list: write output: %v\n", err)
		return 1
	}

	return 0
}

func runListsAddBook(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("lists add-book", flag.ContinueOnError)

	serverURL := serverFlag(flags)
	outputValue := outputFlag(flags, outputFormatText, mutationFormatsHelp)
	listRef := flags.String("list", "", "List name, ID or id:N")
	bookRef := flags.String("book", "", "Book title, ID or id:N")

	help := commandHelp{
		name:        "bookist lists add-book",
		usage:       "bookist lists add-book [options]",
		description: "Add a book to a list",
		details:     []string{referenceHelp, membershipAddOutputHelp},
		examples:    []string{`bookist lists add-book --list "Best Books" --book id:12`},
	}
	if ok, exitCode := parseFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
	}
	if !rejectArgs(flags, "lists add-book", stderr, help) {
		return 2
	}

	if strings.TrimSpace(*listRef) == "" {
		_, _ = fmt.Fprintln(stderr, "--list is required")
		return 2
	}

	if strings.TrimSpace(*bookRef) == "" {
		_, _ = fmt.Fprintln(stderr, "--book is required")
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

	listID, err := resolveListID(ctx, c, *listRef)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	bookID, err := resolveBookID(ctx, c, *bookRef)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	if err := c.Lists.AddBook(ctx, listID, bookID); err != nil {
		_, _ = fmt.Fprintf(stderr, "add book to list: %v\n", err)
		return 1
	}

	text := fmt.Sprintf("added book %d to list %d", bookID, listID)
	if err := writeMutationOutput(stdout, format, addedToListResult{ListID: listID, BookID: bookID, Added: true}, text); err != nil {
		_, _ = fmt.Fprintf(stderr, "add book to list: write output: %v\n", err)
		return 1
	}

	return 0
}

func runListsRM(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("lists rm", flag.ContinueOnError)

	serverURL := serverFlag(flags)
	outputValue := outputFlag(flags, outputFormatText, mutationFormatsHelp)

	help := commandHelp{
		name:        "bookist lists rm",
		usage:       "bookist lists rm [options] <name-or-ID>",
		description: "Remove a book list",
		details:     []string{referenceHelp, removedOutputHelp},
	}
	if ok, exitCode := parseFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
	}

	if flags.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "Error: lists rm requires exactly one name or ID")
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

	listID, err := resolveListID(ctx, c, flags.Arg(0))
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	if err := c.Lists.Delete(ctx, listID); err != nil {
		_, _ = fmt.Fprintf(stderr, "remove list: %v\n", err)
		return 1
	}

	text := fmt.Sprintf("removed list %d", listID)
	if err := writeMutationOutput(stdout, format, removedResult{ID: listID, Removed: true}, text); err != nil {
		_, _ = fmt.Fprintf(stderr, "remove list: write output: %v\n", err)
		return 1
	}

	return 0
}

func runListsRMBook(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("lists rm-book", flag.ContinueOnError)

	serverURL := serverFlag(flags)
	outputValue := outputFlag(flags, outputFormatText, mutationFormatsHelp)
	listRef := flags.String("list", "", "List name, ID or id:N")
	bookRef := flags.String("book", "", "Book title, ID or id:N")

	help := commandHelp{
		name:        "bookist lists rm-book",
		usage:       "bookist lists rm-book [options]",
		description: "Remove a book from a list",
		details:     []string{referenceHelp, membershipRemoveOutputHelp},
	}

	if ok, exitCode := parseFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
	}

	if !rejectArgs(flags, "lists rm-book", stderr, help) {
		return 2
	}

	if strings.TrimSpace(*listRef) == "" {
		_, _ = fmt.Fprintln(stderr, "--list is required")
		return 2
	}

	if strings.TrimSpace(*bookRef) == "" {
		_, _ = fmt.Fprintln(stderr, "--book is required")
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

	listID, err := resolveListID(ctx, c, *listRef)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	bookID, err := resolveBookID(ctx, c, *bookRef)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	if err := c.Lists.RemoveBook(ctx, listID, bookID); err != nil {
		_, _ = fmt.Fprintf(stderr, "remove book from list: %v\n", err)
		return 1
	}

	text := fmt.Sprintf("removed book %d from list %d", bookID, listID)
	if err := writeMutationOutput(stdout, format, removedFromListResult{ListID: listID, BookID: bookID, Removed: true}, text); err != nil {
		_, _ = fmt.Fprintf(stderr, "remove book from list: write output: %v\n", err)
		return 1
	}

	return 0
}

func resolveListID(ctx context.Context, c *client.Client, value string) (int64, error) {
	return resolveReference(ctx, referenceSource{
		kind: "list",
		get: func(ctx context.Context, id int64) (record, error) {
			list, err := c.Lists.Get(ctx, id)
			return record{ID: list.ID, Name: list.Name}, err
		},
		find: func(ctx context.Context, name string) ([]record, error) {
			existing, err := c.Lists.List(ctx, name)
			if err != nil {
				return nil, err
			}

			var matches []record
			for _, list := range existing {
				if strings.EqualFold(list.Name, name) {
					matches = append(matches, record{ID: list.ID, Name: list.Name})
				}
			}

			return matches, nil
		},
	}, value)
}
