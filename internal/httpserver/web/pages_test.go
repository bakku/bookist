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
	if !bytes.Contains(resp.Body.Bytes(), []byte(`class="image is-3by4 has-background-primary-soft"`)) {
		t.Fatalf("expected index response to contain a theme-aware cover background, got %s", resp.Body.String())
	}
	if !bytes.Contains(resp.Body.Bytes(), []byte(`class="is-overlay is-flex is-align-items-center is-justify-content-center has-text-primary-bold"`)) {
		t.Fatalf("expected index response to position the cover placeholder with Bulma helpers, got %s", resp.Body.String())
	}
	if !bytes.Contains(resp.Body.Bytes(), []byte(`data-book-grid class="columns is-multiline"`)) {
		t.Fatalf("expected index response to render the book grid, got %s", resp.Body.String())
	}
	if !bytes.Contains(resp.Body.Bytes(), []byte(`class="column is-half-tablet is-one-third-desktop is-one-quarter-fullhd is-flex"`)) {
		t.Fatalf("expected index response to render responsive book columns, got %s", resp.Body.String())
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
		`aria-label="Primary"`,
		`aria-label="Mobile primary"`,
		`id="mobile-sidebar" popover class="mobile-sidebar has-text-inherit"`,
		`popovertarget="mobile-sidebar" aria-label="Open navigation"`,
		`class="navbar has-shadow app-header" role="banner"`,
		`<strong class="is-size-5 ml-2">Bookist</strong>`,
		`href="#icon-books"`,
		`data-desktop-sidebar class="column is-narrow is-hidden-touch app-sidebar"`,
		`class="columns is-gapless mb-0 app-layout"`,
		`class="navbar-burger"`,
		`class="menu"`,
		`class="notification is-primary is-light"`,
		`class="tag is-primary is-light is-rounded"`,
		`class="is-flex is-align-items-center is-gap-1 is-active"`,
		`aria-current="page"`,
	} {
		if !strings.Contains(body, expected) {
			t.Errorf("expected index response to contain %q", expected)
		}
	}
	if got := strings.Count(body, `<strong class="is-size-5 ml-2">Bookist</strong>`); got != 2 {
		t.Errorf("expected Bookist branding in the header and mobile sidebar, got %d instances", got)
	}
	if strings.Contains(body, `>Navigation</strong>`) {
		t.Error("expected the mobile sidebar to use Bookist branding instead of a navigation heading")
	}
	if strings.Contains(body, `href="/authors"`) || strings.Contains(body, `href="/lists/`) {
		t.Error("expected navigation to hide destinations without web pages")
	}
}

// ── Static Assets ─────────────────────────────────────────────────────────────

func TestStaticStylesheetsAreServed(t *testing.T) {
	app := newTestApp(t)

	tests := []struct {
		path        string
		contentType string
		contains    []byte
	}{
		{path: "/static/bulma.min.css", contentType: "text/css", contains: []byte("bulma.io v1.0.4")},
		{path: "/static/app.css", contentType: "text/css", contains: []byte(".column.app-sidebar")},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			resp := httptest.NewRecorder()
			app.handler.ServeHTTP(resp, req)
			if resp.Code != http.StatusOK {
				t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
			}
			if got := resp.Header().Get("Content-Type"); !strings.HasPrefix(got, tt.contentType) {
				t.Fatalf("expected content type %q, got %q", tt.contentType, got)
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
