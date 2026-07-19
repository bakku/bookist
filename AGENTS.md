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
- Use Pico.css for the web UI styling.
- Prefer simple, testable server and CLI boundaries.
- Add tests where practical, especially for book-management behavior, HTTP handlers, and CLI/server interaction logic.

## Current Stack Decisions

- Prefer the Go standard library first: `net/http`, `http.ServeMux`, `html/template`, `embed`, `database/sql`, and `encoding/json`.
- Do not add a web framework yet. Consider `chi` only if route groups, middleware, nested resources, auth, or larger API structure make the standard mux awkward.
- Use SQLite for persistence with the pure-Go `modernc.org/sqlite` driver.
- Use `github.com/pressly/goose/v3` for embedded SQL migrations. Serving runs migrations automatically at startup.
- Keep Pico.css vendored as an embedded static asset rather than loading it from a CDN.
- Every table, including relationship tables, uses a UUID `TEXT` primary key plus `created_at` and `updated_at` timestamps. ISBN remains nullable in Go and SQLite.
- Collections default to `updated_at DESC, id ASC`; relationship-backed collections use the relationship row's timestamps and ID.
- Enforce durable invariants in both services and SQLite constraints.
- Rewriting the initial migration is acceptable while the schema is pre-release and disposable; once compatibility matters, migrations are append-only.

## API, Web, And CLI Direction

- The web UI should be responsive and comfortable to use from a smartphone.
- The web UI currently lists books only. Do not add form-based book creation unless explicitly requested.
- `POST /api/books` should accept JSON only. The API is the automation boundary for CLI and future skill-based workflows.
- CLI resource-management commands should talk to the HTTP JSON API, not directly to SQLite. Direct DB access is acceptable for `serve` and migration setup.
- Keep the CLI command shape simple for now: `serve`, `migrate`, and resource commands such as `books list` / `books add`.

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
