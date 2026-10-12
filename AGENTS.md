# Bookist

Bookist is intended to be a Go application for managing a home library.

## Product Direction

- Provide a book management application for a personal home library.
- Include a server with a web interface.
- Include a CLI that can connect to the server/web interface and manage books from the terminal.
- Keep the CLI suitable for automation and future skill-based workflows.

## Technical Direction

- Use Go for the application.
- Embed static assets in the binary.
- Keep the web interface server-rendered with native `html/template` templates.
- Use a hand-written, framework-free Bookist stylesheet with a warm, Goodreads-inspired look: cream and dark-brown palette, Merriweather for headings and book titles, Lato for body text.
- Keep web styling build-free, without a frontend toolchain or runtime.
- Implement reusable UI patterns with straightforward native Go template definitions rather than a frontend component framework.
- Prefer simple, testable server and CLI boundaries.
- Add tests where practical, especially for book-management behavior, HTTP handlers, and CLI/server interaction logic.

## Current Stack Decisions

- Prefer the Go standard library first: `net/http`, `http.ServeMux`, `html/template`, `embed`, `database/sql`, and `encoding/json`.
- Do not add a web framework yet. Consider `chi` only if route groups, middleware, nested resources, auth, or larger API structure make the standard mux awkward.
- Use SQLite for persistence with the pure-Go `modernc.org/sqlite` driver.
- Use `github.com/pressly/goose/v3` for embedded SQL migrations. Migrations run through `goose.NewProvider` with `goose.WithSlog` (no goose global state), so output goes to the application logger. Serving runs migrations automatically at startup.
- Keep the Bookist stylesheet and the Merriweather/Lato WOFF2 fonts as embedded static assets. Production requires only the compiled Go binary.
- Every table, including relationship tables, uses an SQLite-generated `INTEGER PRIMARY KEY` represented as `int64` in Go, plus `created_at` and `updated_at` timestamps. ISBN remains nullable in Go and SQLite.
- Collections default to `updated_at DESC, id ASC`; relationship-backed collections use the relationship row's timestamps and ID.
- Exception: a book's authors are ordered by `book_authors.position` (unique per book, set from the order of `author_ids`), because author order is data rather than edit history. The books package is the only reader and writer of `book_authors`.
- Enforce durable invariants in both services and SQLite constraints.
- Rewriting the initial migration is acceptable while the schema is pre-release and disposable; once compatibility matters, migrations are append-only.

## API, Web, And CLI Direction

- The web UI should be responsive and comfortable to use from a smartphone.
- The web UI currently renders the My Books page (list and cover views, shelves sidebar, reading stats) from static sample data in `internal/httpserver/web/sample_library.go`. Wire it to the books, reads, and lists backends incrementally, replacing the sample data. Do not add form-based book creation unless explicitly requested.
- `POST /api/books` should accept JSON only. The API is the automation boundary for CLI and future skill-based workflows.
- API errors are JSON: `{"error": {"code", "message", "field"?}}`. Domain sentinels are built with `internal/apperr` (kinds map to 404/400/409 in one place in `api`); errors shared across packages live in `internal/domain`. Unknown errors become a generic 500.
- CLI resource-management commands should talk to the HTTP JSON API, not directly to SQLite. Direct DB access is acceptable for `serve` and migration setup.
- `internal/app` owns startup: it opens and migrates the database, wires repositories and services, binds the listener (`net.Listen`) before logging the address, and serves until its context is cancelled, then calls `Shutdown` with a timeout and closes the database. `serve` (via `signal.NotifyContext` for SIGINT/SIGTERM) and `migrate` only parse flags and call it. Server timeouts are sized for cover uploads (about 15 MiB JSON bodies).
- `serve`/`migrate` settings: `--addr` (`$BOOKIST_ADDR`, default `:8080`), `--data-dir` (`$BOOKIST_DATA_DIR`, default `./data`) and `--db` (`$BOOKIST_DB`, default `<data-dir>/bookist.db`); flags override env, blank env values are ignored. Startup logs the resolved absolute data dir and database path.
- Keep the CLI command shape simple for now: `serve`, `migrate`, and resource commands such as `books ls` / `books add`. `list` is accepted as an alias of `ls` for every resource.
- The CLI server URL defaults to `$BOOKIST_SERVER` (blank is ignored), falling back to `http://localhost:8080`; `--server` overrides it. The env lookup is the package variable `lookupEnv`, replaced in tests via `SetLookupEnv` (`export_test.go`).
- CLI output format is `--output` / `-o` (never `--format`, which means the book format in `books add`/`edit`). `ls` and `show` accept `tsv|pretty|json` (default `pretty`); `add`, `edit`, `rm` and `lists add-book`/`rm-book` accept `text|json` (default `text`). With `-o json`, add/edit print the API's created/updated record; `rm` prints `{"id": N, "removed": true}`; `lists add-book`/`rm-book` print `{"list_id": N, "book_id": N, "added"|"removed": true}`. `books ls` has an AUTHORS column (comma-separated names).
- CLI flags may appear before, between or after positional arguments (`parseFlags` reorders them). Commands that take no positional arguments reject stray ones with a usage error (exit 2).
- CLI book, author and list references are exact titles/names or IDs, resolved in one place (`resolveReference`). `id:N` always means ID N and must exist. A plain positive integer is looked up as both an ID and an exact title/name; matching different records is an ambiguity error naming each candidate and suggesting `id:N`. Reads take plain read IDs.
- `books add --author` requires existing authors; `--create-authors` opts in to creating missing ones and reports each on stderr. `books edit --author` only accepts existing authors.

## Testing Notes

- Prefer tests that exercise the real SQLite repository against a migrated temporary database instead of mocks when practical.
- For create behavior, verify what was actually persisted in SQLite, not only what the called method returned.
- For list/read behavior, seed SQLite directly and then verify the unit under test reads it correctly.
- HTTP endpoint tests should not depend on another endpoint, the service, or the repository for setup/assertions. Seed and assert database state directly with test helpers.
- Keep endpoint tests focused on one route/behavior at a time; avoid combined create-then-list endpoint tests.
- Group related tests with labeled section markers such as `// ── Reads List ────────────────────────────────────────────────────────────────`.

## Operational Notes

- When running a demo server for external access, bind to `0.0.0.0`, not `127.0.0.1`.
- For long-running manual smoke servers, prefer a built binary over `go run` so the tracked PID is the actual server process.

## Collaboration Notes

- Preserve this direction for future sessions unless the user explicitly changes it.
- Make small, incremental changes and verify them with Go tests when code is added.
- Do not add AI attribution to commits or pull requests: no `Co-Authored-By` trailers for AI assistants and no "Generated with" lines in commit messages or PR descriptions.
