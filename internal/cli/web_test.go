package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"bakku.dev/bookist/internal/cli"
)

// ── Serve ─────────────────────────────────────────────────────────────────────

func TestServeWritesListeningStatusToStdout(t *testing.T) {
	addr := "127.0.0.1:not-a-port"
	dbPath := filepath.Join(t.TempDir(), "bookist.db")
	dataDir := filepath.Join(t.TempDir(), "data")
	var stdout, stderr strings.Builder

	exitCode := cli.Run([]string{"serve", "--addr", addr, "--db", dbPath, "--data-dir", dataDir}, &stdout, &stderr)

	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if got, want := stdout.String(), "Bookist listening on "+addr+"\n"; got != want {
		t.Fatalf("expected stdout %q, got %q", want, got)
	}
	if info, err := os.Stat(filepath.Join(dataDir, "book-covers")); err != nil || !info.IsDir() {
		t.Fatalf("expected book-covers directory, got info=%v err=%v", info, err)
	}
}
