package cli_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// ── Serve ─────────────────────────────────────────────────────────────────────

func TestServeReportsListenFailureWithoutAnnouncingStartup(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "data")
	setEnv(t, nil)

	exitCode, stdout, stderr := runCLI([]string{"serve", "--addr", "127.0.0.1:not-a-port", "--data-dir", dataDir})

	if exitCode != 1 {
		t.Fatalf("expected exit code 1, got %d", exitCode)
	}
	if stdout != "" {
		t.Fatalf("expected empty stdout, got %q", stdout)
	}
	if !strings.Contains(stderr, "listen on 127.0.0.1:not-a-port") {
		t.Fatalf("expected listen error, got %q", stderr)
	}
	if strings.Contains(stderr, "listening") {
		t.Fatalf("expected no listening message, got %q", stderr)
	}
	assertIsDir(t, filepath.Join(dataDir, "book-covers"))
	assertIsFile(t, filepath.Join(dataDir, "bookist.db"))
}

func TestServeLogsResolvedStoragePaths(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "data")
	setEnv(t, nil)

	_, _, stderr := runCLI([]string{"serve", "--addr", "127.0.0.1:not-a-port", "--data-dir", dataDir})

	for _, expected := range []string{"data_dir=" + dataDir, "db=" + filepath.Join(dataDir, "bookist.db")} {
		if !strings.Contains(stderr, expected) {
			t.Errorf("expected stderr to contain %q, got %q", expected, stderr)
		}
	}
}

func TestServeDefaultsToAddrEnvironmentVariable(t *testing.T) {
	setEnv(t, map[string]string{
		"BOOKIST_ADDR":     "127.0.0.1:env-port",
		"BOOKIST_DATA_DIR": filepath.Join(t.TempDir(), "data"),
	})

	exitCode, _, stderr := runCLI([]string{"serve"})

	if exitCode != 1 || !strings.Contains(stderr, "listen on 127.0.0.1:env-port") {
		t.Fatalf("expected listen failure on the BOOKIST_ADDR address, got exit=%d stderr=%q", exitCode, stderr)
	}
}

func TestServeAddrFlagOverridesEnvironmentVariable(t *testing.T) {
	setEnv(t, map[string]string{
		"BOOKIST_ADDR":     "127.0.0.1:env-port",
		"BOOKIST_DATA_DIR": filepath.Join(t.TempDir(), "data"),
	})

	exitCode, _, stderr := runCLI([]string{"serve", "--addr", "127.0.0.1:flag-port"})

	if exitCode != 1 || !strings.Contains(stderr, "listen on 127.0.0.1:flag-port") {
		t.Fatalf("expected listen failure on the --addr address, got exit=%d stderr=%q", exitCode, stderr)
	}
}

// ── Migrate ───────────────────────────────────────────────────────────────────

func TestMigrateCreatesDatabaseInDataDirectory(t *testing.T) {
	dataDir := filepath.Join(t.TempDir(), "data")
	setEnv(t, nil)

	exitCode, _, stderr := runCLI([]string{"migrate", "--data-dir", dataDir})

	if exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d: %s", exitCode, stderr)
	}
	assertIsFile(t, filepath.Join(dataDir, "bookist.db"))
}

func TestMigrateDefaultsToStorageEnvironmentVariables(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "env-data")
	dbPath := filepath.Join(root, "elsewhere", "env.db")

	t.Run("data dir", func(t *testing.T) {
		setEnv(t, map[string]string{"BOOKIST_DATA_DIR": dataDir})

		if exitCode, _, stderr := runCLI([]string{"migrate"}); exitCode != 0 {
			t.Fatalf("expected exit code 0, got %d: %s", exitCode, stderr)
		}
		assertIsFile(t, filepath.Join(dataDir, "bookist.db"))
	})

	t.Run("database path", func(t *testing.T) {
		setEnv(t, map[string]string{"BOOKIST_DATA_DIR": dataDir, "BOOKIST_DB": dbPath})

		if exitCode, _, stderr := runCLI([]string{"migrate"}); exitCode != 0 {
			t.Fatalf("expected exit code 0, got %d: %s", exitCode, stderr)
		}
		assertIsFile(t, dbPath)
	})
}

func TestMigrateFlagsOverrideStorageEnvironmentVariables(t *testing.T) {
	root := t.TempDir()
	envDataDir := filepath.Join(root, "env-data")
	flagDataDir := filepath.Join(root, "flag-data")
	flagDBPath := filepath.Join(root, "flag.db")
	setEnv(t, map[string]string{"BOOKIST_DATA_DIR": envDataDir, "BOOKIST_DB": filepath.Join(root, "env.db")})

	if exitCode, _, stderr := runCLI([]string{"migrate", "--data-dir", flagDataDir}); exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d: %s", exitCode, stderr)
	}
	if exitCode, _, stderr := runCLI([]string{"migrate", "--db", flagDBPath}); exitCode != 0 {
		t.Fatalf("expected exit code 0, got %d: %s", exitCode, stderr)
	}

	assertIsFile(t, filepath.Join(root, "env.db"))
	assertIsFile(t, flagDBPath)
	if _, err := os.Stat(envDataDir); !os.IsNotExist(err) {
		t.Fatalf("expected %s not to be created, got err=%v", envDataDir, err)
	}
}

func TestMigrateLogsToApplicationLogger(t *testing.T) {
	setEnv(t, nil)

	_, _, stderr := runCLI([]string{"migrate", "--data-dir", filepath.Join(t.TempDir(), "data")})

	if !strings.Contains(stderr, "level=INFO") || !strings.Contains(stderr, "applied migration") {
		t.Fatalf("expected structured migration log lines, got %q", stderr)
	}
}

func assertIsDir(t *testing.T, path string) {
	t.Helper()

	if info, err := os.Stat(path); err != nil || !info.IsDir() {
		t.Fatalf("expected directory %s, got info=%v err=%v", path, info, err)
	}
}

func assertIsFile(t *testing.T, path string) {
	t.Helper()

	if info, err := os.Stat(path); err != nil || !info.Mode().IsRegular() {
		t.Fatalf("expected file %s, got info=%v err=%v", path, info, err)
	}
}
