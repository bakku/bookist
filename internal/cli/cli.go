package cli

import (
	"fmt"
	"io"
	"os"
)

const (
	defaultAddr      = ":8080"
	defaultDBPath    = "bookist.db"
	defaultServerURL = "http://localhost:8080"
)

func Run(args []string, stdout io.Writer, stderr io.Writer) int {
	if len(args) == 0 {
		args = []string{"serve"}
	}

	switch args[0] {
	case "serve":
		return runServe(args[1:], stdout, stderr)

	case "migrate":
		return runMigrate(args[1:], stderr)

	case "books":
		return runBooks(args[1:], stdout, stderr)

	case "authors":
		return runAuthors(args[1:], stdout, stderr)

	case "lists":
		return runLists(args[1:], stdout, stderr)

	case "help", "-h", "--help":
		printUsage(stdout)
		return 0

	default:
		_, _ = fmt.Fprintf(stderr, "unknown command %q\n\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func printUsage(w io.Writer) {
	program := "bookist"

	if len(os.Args) > 0 {
		program = os.Args[0]
	}
	_, _ = fmt.Fprintf(w, `Usage:
  %[1]s serve [--addr :8080] [--db bookist.db]
  %[1]s migrate [--db bookist.db]
  %[1]s books list [--server http://localhost:8080]
  %[1]s books add --title TITLE [--isbn ISBN] [--author NAME_OR_ID ...] [--server http://localhost:8080]
  %[1]s authors list [--server http://localhost:8080]
  %[1]s authors add --name NAME [--server http://localhost:8080]
  %[1]s lists list [--server http://localhost:8080]
  %[1]s lists add --name NAME [--description DESC] [--server http://localhost:8080]
  %[1]s lists add-book --list NAME_OR_ID --book TITLE_OR_ID [--server http://localhost:8080]
`, program)
}
