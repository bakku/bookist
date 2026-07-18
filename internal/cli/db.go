package cli

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"io"

	"bakku.dev/bookist/internal/database"
)

func runMigrate(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("migrate", flag.ContinueOnError)

	dbPath := flags.String("db", defaultDBPath, "SQLite database path")

	help := commandHelp{
		name:        "bookist migrate",
		usage:       "bookist migrate [options]",
		description: "Run database migrations",
	}
	if ok, exitCode := parseFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
	}

	db, err := openAndMigrate(context.Background(), *dbPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	defer func() {
		_ = db.Close()
	}()

	return 0
}

func openAndMigrate(ctx context.Context, dbPath string) (*sql.DB, error) {
	db, err := database.Open(ctx, dbPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := database.MigrateUp(db); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate database: %w", err)
	}

	return db, nil
}
