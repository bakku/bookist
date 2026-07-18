package httpserver_test

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"bakku.dev/bookist/internal/testsupport"
)

// ── Index ─────────────────────────────────────────────────────────────────────

func TestIndexListsBooks(t *testing.T) {
	app := newTestApp(t)
	testsupport.InsertBookRow(t, app.db, "Kindred", nil)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()
	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}
	if !bytes.Contains(resp.Body.Bytes(), []byte("Kindred")) {
		t.Fatalf("expected index response to contain book title, got %s", resp.Body.String())
	}
}
