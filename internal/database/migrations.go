package database

import (
	"context"
	"database/sql"
	"embed"
	"fmt"
	"io/fs"
	"log/slog"
	"path/filepath"

	"github.com/pressly/goose/v3"
)

//go:embed migrations/*.sql
var migrations embed.FS

// MigrateUp applies pending migrations and logs each one to logger. Goose's
// own verbose output is left off because it logs every SQL statement.
func MigrateUp(ctx context.Context, db *sql.DB, logger *slog.Logger) error {
	migrationsFS, err := fs.Sub(migrations, "migrations")
	if err != nil {
		return err
	}

	provider, err := goose.NewProvider(goose.DialectSQLite3, db, migrationsFS, goose.WithSlog(logger))
	if err != nil {
		return fmt.Errorf("create migration provider: %w", err)
	}

	results, err := provider.Up(ctx)
	for _, result := range results {
		logger.InfoContext(ctx, "applied migration",
			"version", result.Source.Version,
			"source", filepath.Base(result.Source.Path),
			"duration", result.Duration,
		)
	}

	return err
}
