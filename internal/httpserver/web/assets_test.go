package web_test

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

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
