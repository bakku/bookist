package web_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

// ── Book Covers ───────────────────────────────────────────────────────────────

func TestBookCoverRouteRejectsUnmanagedNames(t *testing.T) {
	app := newTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/book-covers/not-managed.png", nil)
	resp := httptest.NewRecorder()

	app.handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusNotFound {
		t.Fatalf("expected status %d, got %d", http.StatusNotFound, resp.Code)
	}
}

func TestBookCoverRouteServesManagedFile(t *testing.T) {
	app := newTestApp(t)

	key := "0123456789abcdef0123456789abcdef.png"
	image := []byte("\x89PNG\r\n\x1a\ncover")
	if err := os.WriteFile(filepath.Join(app.coverDir, key), image, 0o600); err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/book-covers/"+key, nil)
	resp := httptest.NewRecorder()

	app.handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}

	if got := resp.Header().Get("Content-Type"); got != "image/png" {
		t.Fatalf("expected image/png, got %q", got)
	}

	served, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}

	if !bytes.Equal(served, image) {
		t.Fatalf("expected served bytes %v, got %v", image, served)
	}
}
