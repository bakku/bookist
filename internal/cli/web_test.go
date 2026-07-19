package cli_test

import (
	"path/filepath"
	"strings"
	"testing"

	"bakku.dev/bookist/internal/cli"
)

// ── Serve ─────────────────────────────────────────────────────────────────────

func TestServeWritesListeningStatusToStdout(t *testing.T) {
	addr := "127.0.0.1:not-a-port"
	dbPath := filepath.Join(t.TempDir(), "bookist.db")
	var stdout, stderr strings.Builder

	exitCode := cli.Run([]string{"serve", "--addr", addr, "--db", dbPath}, &stdout, &stderr)

	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if got, want := stdout.String(), "Bookist listening on "+addr+"\n"; got != want {
		t.Fatalf("expected stdout %q, got %q", want, got)
	}
}
