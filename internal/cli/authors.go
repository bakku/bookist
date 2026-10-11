package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/client"
)

func runAuthors(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		_, _ = fmt.Fprintln(stderr, "Error: missing authors command")
		_, _ = fmt.Fprintln(stderr)
		printAuthorsHelp(stderr)
		return 2
	}

	switch args[0] {
	case "ls", "list":
		return runAuthorsLS(args[1:], stdout, stderr)

	case "show":
		return runAuthorsShow(args[1:], stdout, stderr)

	case "add":
		return runAuthorsAdd(args[1:], stdout, stderr)

	case "edit":
		return runAuthorsEdit(args[1:], stdout, stderr)

	case "rm":
		return runAuthorsRM(args[1:], stdout, stderr)

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
			{name: "ls, list", description: "List authors"},
			{name: "show", description: "Show an author"},
			{name: "add", description: "Add an author"},
			{name: "edit", description: "Edit an author"},
			{name: "rm", description: "Remove an author"},
		},
	}, nil)
}

func runAuthorsEdit(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("authors edit", flag.ContinueOnError)
	serverURL := serverFlag(flags)
	outputValue := outputFlag(flags, outputFormatText, mutationFormatsHelp)
	var name optionalStringFlag
	flags.Var(&name, "name", "Author name")
	help := commandHelp{
		name:        "bookist authors edit",
		usage:       "bookist authors edit <name-or-ID> [options]",
		description: "Edit an author",
		details: []string{
			"Omitted fields are unchanged; supplied values replace fields. The required author name cannot be cleared.",
			referenceHelp,
			mutationOutputHelp,
		},
		examples: []string{`bookist authors edit 6 --name "Ursula K. Le Guin"`},
	}
	if ok, exitCode := parseFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
	}
	if flags.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "Error: authors edit requires exactly one name or ID")
		_, _ = fmt.Fprintln(stderr)
		printCommandHelp(stderr, help, flags)
		return 2
	}
	request := authors.UpdateAuthorRequest{Name: supplied(name.value)}
	if isEmptyUpdate(request) {
		_, _ = fmt.Fprintln(stderr, "Error: authors edit requires at least one change")
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
	authorID, err := resolveAuthorID(ctx, c, flags.Arg(0))
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	updated, err := c.Authors.Update(ctx, authorID, request)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "edit author: %v\n", err)
		return 1
	}
	text := fmt.Sprintf("%d\t%s", updated.ID, updated.Name)
	if err := writeMutationOutput(stdout, format, updated, text); err != nil {
		_, _ = fmt.Fprintf(stderr, "edit author: write output: %v\n", err)
		return 1
	}

	return 0
}

func runAuthorsLS(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("authors ls", flag.ContinueOnError)

	serverURL := serverFlag(flags)
	outputValue := outputFlag(flags, outputFormatPretty, outputFormatsHelp)
	query := flags.String("query", "", "Filter authors by name")

	help := commandHelp{
		name:        "bookist authors ls",
		usage:       "bookist authors ls [options]",
		description: "List authors",
	}
	if ok, exitCode := parseFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
	}
	if !rejectArgs(flags, "authors ls", stderr, help) {
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

	listed, err := c.Authors.List(context.Background(), *query)
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

func runAuthorsShow(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("authors show", flag.ContinueOnError)

	serverURL := serverFlag(flags)
	outputValue := outputFlag(flags, outputFormatPretty, outputFormatsHelp)

	help := commandHelp{
		name:        "bookist authors show",
		usage:       "bookist authors show <name-or-ID> [options]",
		description: "Show an author",
		details:     []string{referenceHelp},
		examples: []string{
			"bookist authors show 6",
			"bookist authors show id:6",
			`bookist authors show "Ursula K. Le Guin" --output json`,
		},
	}
	if ok, exitCode := parseFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
	}

	if flags.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "Error: authors show requires exactly one name or ID")
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

	authorID, err := resolveAuthorID(ctx, c, flags.Arg(0))
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}

	author, err := c.Authors.Get(ctx, authorID)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "show author: %v\n", err)
		return 1
	}

	fields := []showField{
		{"ID", strconv.FormatInt(author.ID, 10)},
		{"Name", author.Name},
		{"Created at", timeValue(author.CreatedAt)},
		{"Updated at", timeValue(author.UpdatedAt)},
	}
	if err := writeShowOutput(stdout, format, author, fields); err != nil {
		_, _ = fmt.Fprintf(stderr, "show author: write output: %v\n", err)
		return 1
	}

	return 0
}

func runAuthorsAdd(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("authors add", flag.ContinueOnError)

	serverURL := serverFlag(flags)
	outputValue := outputFlag(flags, outputFormatText, mutationFormatsHelp)
	name := flags.String("name", "", "Author name")

	help := commandHelp{
		name:        "bookist authors add",
		usage:       "bookist authors add [options]",
		description: "Add an author",
		details:     []string{mutationOutputHelp},
	}
	if ok, exitCode := parseFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
	}
	if !rejectArgs(flags, "authors add", stderr, help) {
		return 2
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

	author, err := c.Authors.Create(context.Background(), authors.CreateAuthorRequest{Name: *name})
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "add author: %v\n", err)
		return 1
	}

	text := fmt.Sprintf("%d\t%s", author.ID, author.Name)
	if err := writeMutationOutput(stdout, format, author, text); err != nil {
		_, _ = fmt.Fprintf(stderr, "add author: write output: %v\n", err)
		return 1
	}

	return 0
}

func runAuthorsRM(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("authors rm", flag.ContinueOnError)

	serverURL := serverFlag(flags)
	outputValue := outputFlag(flags, outputFormatText, mutationFormatsHelp)

	help := commandHelp{
		name:        "bookist authors rm",
		usage:       "bookist authors rm [options] <name-or-ID>",
		description: "Remove an author",
		details:     []string{referenceHelp, removedOutputHelp},
	}

	if ok, exitCode := parseFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
	}

	if flags.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "Error: authors rm requires exactly one name or ID")
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

	authorID, err := resolveAuthorID(ctx, c, flags.Arg(0))
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	if err := c.Authors.Delete(ctx, authorID); err != nil {
		_, _ = fmt.Fprintf(stderr, "remove author: %v\n", err)
		return 1
	}

	text := fmt.Sprintf("removed author %d", authorID)
	if err := writeMutationOutput(stdout, format, removedResult{ID: authorID, Removed: true}, text); err != nil {
		_, _ = fmt.Fprintf(stderr, "remove author: write output: %v\n", err)
		return 1
	}

	return 0
}

// resolveAuthors resolves author references to IDs. A reference that matches
// no author is an error, unless create is set and the reference is not
// id:N, in which case the author is created and reported on stderr.
func resolveAuthors(ctx context.Context, c *client.Client, values []string, create bool, stderr io.Writer) ([]int64, error) {
	result := make([]int64, 0, len(values))

	for _, value := range values {
		value = strings.TrimSpace(value)

		authorID, err := resolveAuthorID(ctx, c, value)

		var missing *notFoundError
		if create && errors.As(err, &missing) && !strings.HasPrefix(value, explicitIDPrefix) {
			created, createErr := c.Authors.Create(ctx, authors.CreateAuthorRequest{Name: value})
			if createErr != nil {
				return nil, fmt.Errorf("create author %q: %v", value, createErr)
			}

			_, _ = fmt.Fprintf(stderr, "created author %q (id %d)\n", created.Name, created.ID)
			authorID, err = created.ID, nil
		}
		if err != nil {
			return nil, err
		}

		result = append(result, authorID)
	}

	return result, nil
}

func resolveAuthorID(ctx context.Context, c *client.Client, value string) (int64, error) {
	return resolveReference(ctx, referenceSource{
		kind: "author",
		get: func(ctx context.Context, id int64) (record, error) {
			author, err := c.Authors.Get(ctx, id)
			return record{ID: author.ID, Name: author.Name}, err
		},
		find: func(ctx context.Context, name string) ([]record, error) {
			existing, err := c.Authors.List(ctx, name)
			if err != nil {
				return nil, err
			}

			var matches []record
			for _, author := range existing {
				if strings.EqualFold(author.Name, name) {
					matches = append(matches, record{ID: author.ID, Name: author.Name})
				}
			}

			return matches, nil
		},
	}, value)
}
