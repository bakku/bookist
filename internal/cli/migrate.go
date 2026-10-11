package cli

import (
	"context"
	"flag"
	"fmt"
	"io"

	"bakku.dev/bookist/internal/app"
)

func runMigrate(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("migrate", flag.ContinueOnError)

	dataDir, dbPath := storageFlags(flags)

	help := commandHelp{
		name:        "bookist migrate",
		usage:       "bookist migrate [options]",
		description: "Run database migrations",
	}
	if ok, exitCode := parseFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
	}
	if !rejectArgs(flags, "migrate", stderr, help) {
		return 2
	}

	if err := app.Migrate(context.Background(), app.Config{DataDir: *dataDir, DBPath: *dbPath}, newLogger(stderr)); err != nil {
		_, _ = fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	return 0
}
