package cli

import (
	"flag"
	"fmt"
	"io"
	"log/slog"
	"os"
	"strings"

	"bakku.dev/bookist/internal/client"
)

const (
	defaultAddr      = ":8080"
	defaultDataDir   = "./data"
	defaultServerURL = "http://localhost:8080"

	serverEnvVar  = "BOOKIST_SERVER"
	addrEnvVar    = "BOOKIST_ADDR"
	dataDirEnvVar = "BOOKIST_DATA_DIR"
	dbEnvVar      = "BOOKIST_DB"
)

// lookupEnv reads environment variables for flag defaults. Tests replace it.
var lookupEnv = os.LookupEnv

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

// envDefault returns the trimmed value of the environment variable name when
// it is set and non-blank, otherwise fallback.
func envDefault(name, fallback string) string {
	if value, ok := lookupEnv(name); ok && strings.TrimSpace(value) != "" {
		return strings.TrimSpace(value)
	}

	return fallback
}

// serverFlag registers --server. Its default is $BOOKIST_SERVER when set and
// non-blank, otherwise defaultServerURL.
func serverFlag(flags *flag.FlagSet) *string {
	return flags.String("server", envDefault(serverEnvVar, defaultServerURL), "Bookist server URL (env: "+serverEnvVar+")")
}

// storageFlags registers --data-dir and --db. Their defaults are
// $BOOKIST_DATA_DIR and $BOOKIST_DB when set and non-blank. An empty database
// path means <data-dir>/bookist.db.
func storageFlags(flags *flag.FlagSet) (dataDir, dbPath *string) {
	dataDir = flags.String("data-dir", envDefault(dataDirEnvVar, defaultDataDir), "Directory for managed application data (env: "+dataDirEnvVar+")")
	dbPath = flags.String("db", envDefault(dbEnvVar, ""), "SQLite database path, <data-dir>/bookist.db when unset (env: "+dbEnvVar+")")

	return dataDir, dbPath
}

// newLogger returns the application logger, which writes text to w.
func newLogger(w io.Writer) *slog.Logger {
	return slog.New(slog.NewTextHandler(w, nil))
}

// newClient returns an API client for serverURL. It reports an invalid URL to
// stderr and returns nil.
func newClient(serverURL string, stderr io.Writer) *client.Client {
	c, err := client.New(serverURL)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "invalid server URL: %v\n", err)
		return nil
	}

	return c
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
