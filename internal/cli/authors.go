package cli

import (
	"context"
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
	case "ls":
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
			{name: "ls", description: "List authors"},
			{name: "show", description: "Show an author"},
			{name: "add", description: "Add an author"},
			{name: "edit", description: "Edit an author"},
			{name: "rm", description: "Remove an author"},
		},
	}, nil)
}

func runAuthorsEdit(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("authors edit", flag.ContinueOnError)
	serverURL := flags.String("server", defaultServerURL, "Bookist server URL")
	var name optionalStringFlag
	flags.Var(&name, "name", "Author name")
	help := commandHelp{
		name:        "bookist authors edit",
		usage:       "bookist authors edit <name-or-ID> [options]",
		description: "Edit an author",
		details:     []string{"Omitted fields are unchanged; supplied values replace fields. The required author name cannot be cleared."},
		examples:    []string{`bookist authors edit 6 --name "Ursula K. Le Guin"`},
	}
	if ok, exitCode := parseEditFlags(flags, args, stdout, stderr, help); !ok {
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

	ctx := context.Background()
	c := newClient(*serverURL, stderr)
	if c == nil {
		return 2
	}
	authorID, found, err := resolveAuthorID(ctx, c, flags.Arg(0))
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	if !found {
		_, _ = fmt.Fprintf(stderr, "author not found: %s\n", flags.Arg(0))
		return 1
	}
	updated, err := c.Authors.Update(ctx, authorID, request)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "edit author: %v\n", err)
		return 1
	}
	_, _ = fmt.Fprintf(stdout, "%d\t%s\n", updated.ID, updated.Name)
	return 0
}

func runAuthorsLS(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("authors ls", flag.ContinueOnError)

	serverURL := flags.String("server", defaultServerURL, "Bookist server URL")
	formatValue := flags.String("format", string(outputFormatPretty), "Output format (tsv|pretty|json)")
	query := flags.String("query", "", "Filter authors by name")

	help := commandHelp{
		name:        "bookist authors ls",
		usage:       "bookist authors ls [options]",
		description: "List authors",
	}
	if ok, exitCode := parseFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
	}

	format, err := parseOutputFormat(*formatValue)
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

	serverURL := flags.String("server", defaultServerURL, "Bookist server URL")
	formatValue := flags.String("format", string(outputFormatPretty), "Output format (tsv|pretty|json)")

	help := commandHelp{
		name:        "bookist authors show",
		usage:       "bookist authors show <name-or-ID> [options]",
		description: "Show an author",
		examples: []string{
			"bookist authors show 6",
			`bookist authors show "Ursula K. Le Guin" --format json`,
		},
	}
	if ok, exitCode := parseEditFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
	}

	if flags.NArg() != 1 {
		_, _ = fmt.Fprintln(stderr, "Error: authors show requires exactly one name or ID")
		_, _ = fmt.Fprintln(stderr)
		printCommandHelp(stderr, help, flags)
		return 2
	}

	format, err := parseOutputFormat(*formatValue)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "Error: %v\n", err)
		return 2
	}

	ctx := context.Background()
	c := newClient(*serverURL, stderr)
	if c == nil {
		return 2
	}

	authorID, found, err := resolveAuthorID(ctx, c, flags.Arg(0))
	if err != nil {
		_, _ = fmt.Fprintln(stderr, err)
		return 1
	}
	if !found {
		_, _ = fmt.Fprintf(stderr, "author not found: %s\n", flags.Arg(0))
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

	serverURL := flags.String("server", defaultServerURL, "Bookist server URL")
	name := flags.String("name", "", "Author name")

	help := commandHelp{
		name:        "bookist authors add",
		usage:       "bookist authors add [options]",
		description: "Add an author",
	}
	if ok, exitCode := parseFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
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

	_, _ = fmt.Fprintf(stdout, "%d\t%s\n", author.ID, author.Name)
	return 0
}

func runAuthorsRM(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("authors rm", flag.ContinueOnError)

	serverURL := flags.String("server", defaultServerURL, "Bookist server URL")

	help := commandHelp{
		name:        "bookist authors rm",
		usage:       "bookist authors rm [options] <name-or-ID>",
		description: "Remove an author",
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

	ctx := context.Background()
	c := newClient(*serverURL, stderr)
	if c == nil {
		return 2
	}

	authorID, found, err := resolveAuthorID(ctx, c, flags.Arg(0))
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	if !found {
		_, _ = fmt.Fprintf(stderr, "author not found: %s\n", flags.Arg(0))
		return 1
	}

	if err := c.Authors.Delete(ctx, authorID); err != nil {
		_, _ = fmt.Fprintf(stderr, "remove author: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "removed author %d\n", authorID)

	return 0
}

func resolveOrCreateAuthors(ctx context.Context, c *client.Client, values []string) ([]int64, error) {
	if len(values) == 0 {
		return nil, nil
	}

	var result []int64

	for _, val := range values {
		val = strings.TrimSpace(val)
		if val == "" {
			continue
		}

		authorID, found, err := resolveAuthorID(ctx, c, val)
		if err != nil {
			return nil, err
		}

		if found {
			result = append(result, authorID)
			continue
		}

		created, err := c.Authors.Create(ctx, authors.CreateAuthorRequest{Name: val})
		if err != nil {
			return nil, fmt.Errorf("create author %q: %v", val, err)
		}

		result = append(result, created.ID)
	}

	return result, nil
}

func resolveExistingAuthors(ctx context.Context, c *client.Client, values []string) ([]int64, error) {
	result := make([]int64, 0, len(values))

	for _, value := range values {
		authorID, found, err := resolveAuthorID(ctx, c, value)
		if err != nil {
			return nil, err
		}
		if !found {
			return nil, fmt.Errorf("author not found: %s", value)
		}

		result = append(result, authorID)
	}

	return result, nil
}

func resolveAuthorID(ctx context.Context, c *client.Client, value string) (int64, bool, error) {
	value = strings.TrimSpace(value)

	id, isID, err := parseIDReference(value)
	if err != nil {
		return 0, false, err
	}

	if isID {
		return id, true, nil
	}

	id, found, err := resolveAuthorName(ctx, c, value)
	if err != nil {
		return 0, false, err
	}

	if !found {
		return 0, false, nil
	}

	return id, true, nil
}

func resolveAuthorName(ctx context.Context, c *client.Client, name string) (int64, bool, error) {
	existing, err := c.Authors.List(ctx, name)
	if err != nil {
		return 0, false, fmt.Errorf("fetch authors: %w", err)
	}

	var matches []authors.Author
	for _, author := range existing {
		if strings.EqualFold(author.Name, name) {
			matches = append(matches, author)
		}
	}

	if len(matches) > 1 {
		return 0, false, fmt.Errorf("author %q exists multiple times; pass an author ID instead", name)
	}

	if len(matches) == 1 {
		return matches[0].ID, true, nil
	}

	return 0, false, nil
}
