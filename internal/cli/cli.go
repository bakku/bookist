package cli

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"os"
	"time"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/database"
	"bakku.dev/bookist/internal/httpserver"
	"bakku.dev/bookist/internal/lists"
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
		fmt.Fprintf(stderr, "unknown command %q\n\n", args[0])
		printUsage(stderr)
		return 2
	}
}

func runServe(args []string, stdout io.Writer, stderr io.Writer) int {
	flags := flag.NewFlagSet("serve", flag.ContinueOnError)
	flags.SetOutput(stderr)
	addr := flags.String("addr", defaultAddr, "HTTP address to listen on")
	dbPath := flags.String("db", defaultDBPath, "SQLite database path")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	ctx := context.Background()
	db, err := database.Open(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open database: %v\n", err)
		return 1
	}
	defer db.Close()

	if err := database.MigrateUp(db); err != nil {
		fmt.Fprintf(stderr, "migrate database: %v\n", err)
		return 1
	}

	authorRepo := authors.NewSQLiteRepository(db)
	authorService := authors.NewService(authorRepo)
	listRepo := lists.NewSQLiteRepository(db)
	listService := lists.NewService(listRepo)
	bookService := books.NewService(books.NewSQLiteRepository(db), authorRepo)
	server, err := httpserver.New(bookService, authorService, listService)
	if err != nil {
		fmt.Fprintf(stderr, "create server: %v\n", err)
		return 1
	}

	fmt.Fprintf(stdout, "Bookist listening on %s\n", *addr)
	httpServer := &http.Server{
		Addr:              *addr,
		Handler:           server.Handler(),
		ReadHeaderTimeout: 5 * time.Second,
		ErrorLog:          slog.NewLogLogger(slog.NewTextHandler(stderr, nil), slog.LevelError),
	}
	if err := httpServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		fmt.Fprintf(stderr, "serve HTTP: %v\n", err)
		return 1
	}

	return 0
}

func runMigrate(args []string, stderr io.Writer) int {
	flags := flag.NewFlagSet("migrate", flag.ContinueOnError)
	flags.SetOutput(stderr)
	dbPath := flags.String("db", defaultDBPath, "SQLite database path")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	ctx := context.Background()
	db, err := database.Open(ctx, *dbPath)
	if err != nil {
		fmt.Fprintf(stderr, "open database: %v\n", err)
		return 1
	}
	defer db.Close()

	if err := database.MigrateUp(db); err != nil {
		fmt.Fprintf(stderr, "migrate database: %v\n", err)
		return 1
	}

	return 0
}

func printUsage(w io.Writer) {
	program := "bookist"
	if len(os.Args) > 0 {
		program = os.Args[0]
	}
	fmt.Fprintf(w, `Usage:
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
