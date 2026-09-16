package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"time"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/covers"
	"bakku.dev/bookist/internal/httpserver"
	"bakku.dev/bookist/internal/lists"
	"bakku.dev/bookist/internal/reads"
)

func runServe(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)

	addr := flags.String("addr", defaultAddr, "HTTP address to listen on")
	dbPath := flags.String("db", defaultDBPath, "SQLite database path")
	dataDir := flags.String("data-dir", defaultDataDir, "Directory for managed application data")

	help := commandHelp{
		name:        "bookist serve",
		usage:       "bookist serve [options]",
		description: "Start the Bookist server",
	}
	if ok, exitCode := parseFlags(flags, args, stdout, stderr, help); !ok {
		return exitCode
	}

	coverStore, err := covers.NewStore(filepath.Join(*dataDir, "book-covers"))
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "initialize cover storage: %v\n", err)
		return 1
	}

	db, err := openAndMigrate(context.Background(), *dbPath)
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "%v\n", err)
		return 1
	}

	defer func() {
		_ = db.Close()
	}()

	authorRepo := authors.NewSQLiteRepository(db)
	authorService := authors.NewService(authorRepo)

	listRepo := lists.NewSQLiteRepository(db)
	listService := lists.NewService(listRepo)

	bookRepo := books.NewSQLiteRepository(db)
	appLogger := slog.New(slog.NewTextHandler(stderr, nil))
	bookService := books.NewServiceWithLogger(bookRepo, authorRepo, coverStore, appLogger)

	readRepo := reads.NewSQLiteRepository(db)
	readService := reads.NewService(readRepo)

	server, err := httpserver.New(bookService, authorService, listService, readService, coverStore)

	if err != nil {
		_, _ = fmt.Fprintf(stderr, "create server: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "Bookist listening on %s\n", *addr)

	httpServer := &http.Server{
		Addr:              *addr,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ErrorLog:          slog.NewLogLogger(appLogger.Handler(), slog.LevelError),
	}

	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		_, _ = fmt.Fprintf(stderr, "serve HTTP: %v\n", err)
		return 1
	}

	return 0
}
