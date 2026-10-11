package web_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ── Static Assets ─────────────────────────────────────────────────────────────

func TestStaticAssetsAreServed(t *testing.T) {
	app := newTestApp(t)

	tests := []struct {
		path        string
		contentType string
		contains    []byte
	}{
		{path: "/static/app.css", contentType: "text/css", contains: []byte(".book-row")},
		{path: "/static/fonts/merriweather-latin-400-normal.woff2", contentType: "font/woff2", contains: []byte("wOF2")},
		{path: "/static/fonts/merriweather-latin-400-italic.woff2", contentType: "font/woff2", contains: []byte("wOF2")},
		{path: "/static/fonts/merriweather-latin-700-normal.woff2", contentType: "font/woff2", contains: []byte("wOF2")},
		{path: "/static/fonts/lato-latin-400-normal.woff2", contentType: "font/woff2", contains: []byte("wOF2")},
		{path: "/static/fonts/lato-latin-700-normal.woff2", contentType: "font/woff2", contains: []byte("wOF2")},
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
				t.Errorf("expected asset to contain %q", tt.contains)
			}
		})
	}
}

func TestStaticAssetsAreCacheableWithETag(t *testing.T) {
	app := newTestApp(t)

	req := httptest.NewRequest(http.MethodGet, "/static/app.css", nil)
	resp := httptest.NewRecorder()
	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d, got %d", http.StatusOK, resp.Code)
	}

	etag := resp.Header().Get("ETag")
	if len(etag) < 3 || !strings.HasPrefix(etag, `"`) || !strings.HasSuffix(etag, `"`) {
		t.Fatalf("expected strong quoted ETag, got %q", etag)
	}
	if got := resp.Header().Get("Cache-Control"); got != "no-cache" {
		t.Errorf("expected Cache-Control %q, got %q", "no-cache", got)
	}

	req = httptest.NewRequest(http.MethodGet, "/static/app.css", nil)
	req.Header.Set("If-None-Match", etag)
	resp = httptest.NewRecorder()
	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusNotModified {
		t.Fatalf("expected status %d, got %d", http.StatusNotModified, resp.Code)
	}
	if resp.Body.Len() != 0 {
		t.Errorf("expected empty body, got %d bytes", resp.Body.Len())
	}
	if got := resp.Header().Get("ETag"); got != etag {
		t.Errorf("expected ETag %q on 304, got %q", etag, got)
	}

	req = httptest.NewRequest(http.MethodGet, "/static/app.css", nil)
	req.Header.Set("If-None-Match", `"stale"`)
	resp = httptest.NewRecorder()
	app.handler.ServeHTTP(resp, req)
	if resp.Code != http.StatusOK {
		t.Fatalf("expected status %d for stale ETag, got %d", http.StatusOK, resp.Code)
	}
}

func TestStaticAssetsReturnNotFoundForMissingFiles(t *testing.T) {
	app := newTestApp(t)

	for _, path := range []string{"/static/missing.css", "/static/fonts", "/static/"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, path, nil)
			resp := httptest.NewRecorder()
			app.handler.ServeHTTP(resp, req)
			if resp.Code != http.StatusNotFound {
				t.Fatalf("expected status %d, got %d", http.StatusNotFound, resp.Code)
			}
		})
	}
}
