package cli

import (
	"fmt"
	"io"
)

const (
	defaultAddr      = ":8080"
	defaultDBPath    = "bookist.db"
	defaultServerURL = "http://localhost:8080"
)

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		printRootHelp(stdout)
		return 0
	}

	switch args[0] {
	case "serve":
		return runServe(args[1:], stdout, stderr)

	case "migrate":
		return runMigrate(args[1:], stdout, stderr)

	case "books":
		return runBooks(args[1:], stdout, stderr)

	case "authors":
		return runAuthors(args[1:], stdout, stderr)

	case "lists":
		return runLists(args[1:], stdout, stderr)

	case "reads":
		return runReads(args[1:], stdout, stderr)

	case "help", "h":
		if len(args) == 1 {
			printRootHelp(stdout)
			return 0
		}
		return Run(append(args[1:], "--help"), stdout, stderr)

	case "-h", "--help":
		printRootHelp(stdout)
		return 0

	default:
		_, _ = fmt.Fprintf(stderr, "Error: unknown command %q\n\n", args[0])
		printRootHelp(stderr)
		return 2
	}
}

func printRootHelp(w io.Writer) {
	printCommandHelp(w, commandHelp{
		name:        "bookist",
		usage:       "bookist [command [command options]]",
		description: "Manage a home library",
		commands: []helpCommand{
			{name: "serve", description: "Start the Bookist server"},
			{name: "migrate", description: "Run database migrations"},
			{name: "books", description: "Manage books"},
			{name: "authors", description: "Manage authors"},
			{name: "lists", description: "Manage book lists"},
			{name: "reads", description: "Manage book reads"},
			{name: "help, h", description: "Show help for a command"},
		},
	}, nil)
}
