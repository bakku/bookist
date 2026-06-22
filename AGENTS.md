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

## Collaboration Notes

- Preserve this direction for future sessions unless the user explicitly changes it.
- Make small, incremental changes and verify them with Go tests when code is added.
