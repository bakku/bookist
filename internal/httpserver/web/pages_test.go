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
	if !bytes.Contains(resp.Body.Bytes(), []byte(`data-book-grid class="columns is-mobile is-multiline book-grid"`)) {
		t.Fatalf("expected index response to render the book grid, got %s", resp.Body.String())
	}
	if !bytes.Contains(resp.Body.Bytes(), []byte(`1 in library`)) {
		t.Fatalf("expected index response to describe the singular book count, got %s", resp.Body.String())
	}
}

// ── Rendering ─────────────────────────────────────────────────────────────────

func TestIndexRendersBulmaAppShell(t *testing.T) {
	app := newTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()
	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}

	body := resp.Body.String()
	for _, expected := range []string{
		`href="/static/bulma.min.css"`,
		`href="/static/app.css"`,
		`class="navbar app-header"`,
		`aria-label="Primary"`,
		`aria-label="Mobile primary"`,
		`class="menu"`,
		`class="notification is-primary is-light mt-5"`,
		`class="tag is-primary is-light is-rounded"`,
		`aria-current="page" class="is-active"`,
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("expected index response to contain %q", expected)
		}
	}
}

// ── Static Assets ─────────────────────────────────────────────────────────────

func TestStaticStylesheetsAreServed(t *testing.T) {
	app := newTestApp(t)

	tests := []struct {
		path     string
		contains []byte
	}{
		{path: "/static/bulma.min.css", contains: []byte("bulma.io v1.0.4")},
		{path: "/static/app.css", contains: []byte(".app-sidebar")},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
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
			if !bytes.Contains(body, tt.contains) {
				t.Errorf("expected stylesheet to contain %q", tt.contains)
			}
		})
	}
}
