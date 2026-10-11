// Package app wires Bookist's storage, services and HTTP server together and
// runs them until the context is cancelled.
package app

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/covers"
	"bakku.dev/bookist/internal/database"
	"bakku.dev/bookist/internal/httpserver"
	"bakku.dev/bookist/internal/lists"
	"bakku.dev/bookist/internal/reads"
)

const (
	// DBFileName is the database file inside the data directory when no
	// explicit database path is configured.
	DBFileName = "bookist.db"

	coverDirName = "book-covers"

	readHeaderTimeout = 5 * time.Second
	// readTimeout covers a cover upload, which can be a JSON body of about
	// 15 MiB sent over a slow connection.
	readTimeout     = 2 * time.Minute
	idleTimeout     = 2 * time.Minute
	shutdownTimeout = 15 * time.Second
)

// Config holds the settings shared by serving and migrating.
type Config struct {
	// Addr is the HTTP address to listen on.
	Addr string
	// DataDir holds managed application data such as book covers.
	DataDir string
	// DBPath is the SQLite database path. It defaults to DBFileName inside
	// DataDir.
	DBPath string
}

// resolved returns a copy of c with DataDir and DBPath made absolute, so they
// no longer depend on the working directory.
func (c Config) resolved() (Config, error) {
	if c.DBPath == "" {
		c.DBPath = filepath.Join(c.DataDir, DBFileName)
	}

	var err error
	if c.DataDir, err = filepath.Abs(c.DataDir); err != nil {
		return Config{}, fmt.Errorf("resolve data directory: %w", err)
	}
	if c.DBPath, err = filepath.Abs(c.DBPath); err != nil {
		return Config{}, fmt.Errorf("resolve database path: %w", err)
	}

	return c, nil
}

// Migrate opens the database, applies pending migrations and closes it.
func Migrate(ctx context.Context, cfg Config, logger *slog.Logger) error {
	cfg, err := cfg.resolved()
	if err != nil {
		return err
	}

	db, err := openAndMigrate(ctx, cfg, logger)
	if err != nil {
		return err
	}

	return db.Close()
}

// App is a Bookist server with its database open and its listener bound. Run
// serves until its context is cancelled and releases everything App holds.
type App struct {
	logger *slog.Logger
	db     *sql.DB
	ln     net.Listener
	server *http.Server
}

// New opens and migrates the database, wires the services and binds the
// listen address. The caller must call Run, which releases the resources
// even when serving fails.
func New(ctx context.Context, cfg Config, logger *slog.Logger) (*App, error) {
	cfg, err := cfg.resolved()
	if err != nil {
		return nil, err
	}

	coverStore, err := covers.NewStore(filepath.Join(cfg.DataDir, coverDirName))
	if err != nil {
		return nil, fmt.Errorf("initialize cover storage: %w", err)
	}

	db, err := openAndMigrate(ctx, cfg, logger)
	if err != nil {
		return nil, err
	}

	authorRepo := authors.NewSQLiteRepository(db)
	authorService := authors.NewService(authorRepo)

	listRepo := lists.NewSQLiteRepository(db)
	listService := lists.NewService(listRepo)

	bookRepo := books.NewSQLiteRepository(db)
	bookService := books.NewServiceWithLogger(bookRepo, authorRepo, coverStore, logger)

	readRepo := reads.NewSQLiteRepository(db)
	readService := reads.NewService(readRepo)

	handler, err := httpserver.New(bookService, authorService, listService, readService, coverStore, logger)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("create server: %w", err)
	}

	ln, err := new(net.ListenConfig).Listen(ctx, "tcp", cfg.Addr)
	if err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("listen on %s: %w", cfg.Addr, err)
	}

	return &App{
		logger: logger,
		db:     db,
		ln:     ln,
		server: &http.Server{
			Handler:           handler.Handler(),
			ReadHeaderTimeout: readHeaderTimeout,
			ReadTimeout:       readTimeout,
			IdleTimeout:       idleTimeout,
			ErrorLog:          slog.NewLogLogger(logger.Handler(), slog.LevelError),
		},
	}, nil
}

// Addr returns the address the server is listening on.
func (a *App) Addr() net.Addr {
	return a.ln.Addr()
}

// Run serves requests until ctx is cancelled, then waits for in-flight
// requests to finish, up to a timeout, and closes the database.
func (a *App) Run(ctx context.Context) error {
	a.logger.Info("listening", "addr", a.ln.Addr().String())

	serveErr := make(chan error, 1)
	go func() {
		serveErr <- a.server.Serve(a.ln)
	}()

	var err error
	select {
	case err = <-serveErr:
		err = fmt.Errorf("serve HTTP: %w", err)
	case <-ctx.Done():
		a.logger.Info("shutting down")

		shutdownCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), shutdownTimeout)
		defer cancel()

		if err = a.server.Shutdown(shutdownCtx); err != nil {
			err = fmt.Errorf("shut down HTTP server: %w", err)
			_ = a.server.Close()
		}
		<-serveErr
	}

	if closeErr := a.db.Close(); closeErr != nil {
		err = errors.Join(err, fmt.Errorf("close database: %w", closeErr))
	}

	if err == nil {
		a.logger.Info("stopped")
	}

	return err
}

func openAndMigrate(ctx context.Context, cfg Config, logger *slog.Logger) (*sql.DB, error) {
	logger.Info("using storage", "data_dir", cfg.DataDir, "db", cfg.DBPath)

	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0o755); err != nil {
		return nil, fmt.Errorf("create database directory: %w", err)
	}

	db, err := database.Open(ctx, cfg.DBPath)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}

	if err := database.MigrateUp(ctx, db, logger); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("migrate database: %w", err)
	}

	return db, nil
}
