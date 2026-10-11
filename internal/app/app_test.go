package app_test

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"net/http"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"bakku.dev/bookist/internal/app"
	"bakku.dev/bookist/internal/testsupport"
)

// ── Run ───────────────────────────────────────────────────────────────────────

func TestRunServesRequestsAndStopsWhenContextIsCancelled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cfg := app.Config{Addr: "127.0.0.1:0", DataDir: filepath.Join(t.TempDir(), "data")}

	server, err := app.New(ctx, cfg, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- server.Run(ctx) }()

	resp, err := http.Get("http://" + server.Addr().String() + "/api/books")
	if err != nil {
		t.Fatal(err)
	}
	body, err := io.ReadAll(resp.Body)
	_ = resp.Body.Close()
	if err != nil {
		t.Fatal(err)
	}
	if resp.StatusCode != http.StatusOK || strings.TrimSpace(string(body)) != "[]" {
		t.Fatalf("expected 200 [], got %d %q", resp.StatusCode, body)
	}

	cancel()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("expected clean shutdown, got %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("Run did not return after the context was cancelled")
	}

	if _, err := http.Get("http://" + server.Addr().String() + "/api/books"); err == nil {
		t.Fatal("expected the listener to be closed")
	}
}

func TestRunClosesDatabaseOnShutdown(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	dataDir := filepath.Join(t.TempDir(), "data")
	server, err := app.New(ctx, app.Config{Addr: "127.0.0.1:0", DataDir: dataDir}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- server.Run(ctx) }()

	cancel()
	if err := <-done; err != nil {
		t.Fatal(err)
	}

	// A closed WAL database checkpoints and removes its -wal file.
	if matches, _ := filepath.Glob(filepath.Join(dataDir, "bookist.db-wal")); len(matches) != 0 {
		t.Fatalf("expected the database to be closed, found %v", matches)
	}
}

// ── New ───────────────────────────────────────────────────────────────────────

func TestNewReportsAddressInUse(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	logger := slog.New(slog.DiscardHandler)
	first, err := app.New(ctx, app.Config{Addr: "127.0.0.1:0", DataDir: filepath.Join(t.TempDir(), "first")}, logger)
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- first.Run(ctx) }()
	defer func() {
		cancel()
		<-done
	}()

	_, err = app.New(ctx, app.Config{Addr: first.Addr().String(), DataDir: filepath.Join(t.TempDir(), "second")}, logger)

	if err == nil || !strings.Contains(err.Error(), "listen on") {
		t.Fatalf("expected a listen error, got %v", err)
	}
}

func TestNewDefaultsDatabaseToDataDirectory(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	dataDir := filepath.Join(t.TempDir(), "data")
	server, err := app.New(ctx, app.Config{Addr: "127.0.0.1:0", DataDir: dataDir}, slog.New(slog.DiscardHandler))
	if err != nil {
		t.Fatal(err)
	}
	done := make(chan error, 1)
	go func() { done <- server.Run(ctx) }()
	cancel()
	<-done

	db := openDB(t, filepath.Join(dataDir, "bookist.db"))
	testsupport.AssertSQLCount(t, db, 1, "SELECT COUNT(*) FROM goose_db_version WHERE version_id = 1")
}

// ── Migrate ───────────────────────────────────────────────────────────────────

func TestMigrateIsIdempotent(t *testing.T) {
	cfg := app.Config{DataDir: filepath.Join(t.TempDir(), "data")}

	for range 2 {
		if err := app.Migrate(context.Background(), cfg, slog.New(slog.DiscardHandler)); err != nil {
			t.Fatal(err)
		}
	}

	db := openDB(t, filepath.Join(cfg.DataDir, "bookist.db"))
	testsupport.AssertSQLCount(t, db, 1, "SELECT COUNT(*) FROM goose_db_version WHERE version_id = 1")
}

func openDB(t *testing.T, path string) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", path)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = db.Close() })

	return db
}
