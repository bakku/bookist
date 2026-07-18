package cli

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"time"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/httpserver"
	"bakku.dev/bookist/internal/lists"
)

func runServe(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)

	addr := flags.String("addr", defaultAddr, "HTTP address to listen on")
	dbPath := flags.String("db", defaultDBPath, "SQLite database path")

	help := commandHelp{
		name:        "bookist serve",
		usage:       "bookist serve [options]",
		description: "Start the Bookist server",
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

	authorRepo := authors.NewSQLiteRepository(db)
	authorService := authors.NewService(authorRepo)

	listRepo := lists.NewSQLiteRepository(db)
	listService := lists.NewService(listRepo)

	bookRepo := books.NewSQLiteRepository(db)
	bookService := books.NewService(bookRepo, authorRepo)

	server, err := httpserver.New(bookService, authorService, listService)

	if err != nil {
		_, _ = fmt.Fprintf(stderr, "create server: %v\n", err)
		return 1
	}

	_, _ = fmt.Fprintf(stdout, "Bookist listening on %s\n", *addr)

	httpServer := &http.Server{
		Addr:              *addr,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ErrorLog:          slog.NewLogLogger(slog.NewTextHandler(stderr, nil), slog.LevelError),
	}

	if err := httpServer.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
		_, _ = fmt.Fprintf(stderr, "serve HTTP: %v\n", err)
		return 1
	}

	return 0
}
