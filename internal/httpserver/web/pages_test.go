package web_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bakku.dev/bookist/internal/testsupport"
)

// ── Index ─────────────────────────────────────────────────────────────────────

func TestIndexListsBooks(t *testing.T) {
	app := newTestApp(t)

	isbn := "9780807083697"

	testsupport.InsertBookRow(t, app.db, "Kindred", &isbn)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()

	app.handler.ServeHTTP(resp, req)

	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}

	if !bytes.Contains(resp.Body.Bytes(), []byte("Kindred")) {
		t.Fatalf("expected index response to contain book title, got %s", resp.Body.String())
	}
	if !bytes.Contains(resp.Body.Bytes(), []byte(isbn)) {
		t.Fatalf("expected index response to contain ISBN, got %s", resp.Body.String())
	}
	if !bytes.Contains(resp.Body.Bytes(), []byte("2026-01-02")) {
		t.Fatalf("expected index response to contain added date, got %s", resp.Body.String())
	}
	if !bytes.Contains(resp.Body.Bytes(), []byte(`aria-label="Cover unavailable for Kindred"`)) {
		t.Fatalf("expected index response to contain a cover placeholder, got %s", resp.Body.String())
	}
	if !bytes.Contains(resp.Body.Bytes(), []byte(`grid-cols-2`)) {
		t.Fatalf("expected index response to render the book grid, got %s", resp.Body.String())
	}
	if !bytes.Contains(resp.Body.Bytes(), []byte(`1 book in library`)) {
		t.Fatalf("expected index response to describe the singular book count, got %s", resp.Body.String())
	}
}

// ── Static Assets ─────────────────────────────────────────────────────────────

func TestStaticAppCSSIsServed(t *testing.T) {
	app := newTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/static/app.css", nil)
	resp := httptest.NewRecorder()
	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}
	if got := resp.Header().Get("Content-Type"); !strings.HasPrefix(got, "text/css") {
		t.Fatalf("expected CSS content type, got %q", got)
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		t.Fatal(err)
	}
	for _, expected := range [][]byte{
		[]byte("tailwindcss v4."),
		[]byte("--bookist-ui-primary-500:#4250af"),
		[]byte("--bookist-ui-muted:oklch(71% 0 271)"),
		[]byte("--bookist-ui-border:oklch(87% 0 none)"),
		[]byte(".sidebar\\:block"),
		[]byte(".bookist-scrollbar"),
		[]byte(".book-cover"),
		[]byte(".ui-card"),
		[]byte(".ui-card__inner"),
	} {
		if !bytes.Contains(body, expected) {
			t.Errorf("expected stylesheet to contain %q", expected)
		}
	}
}
