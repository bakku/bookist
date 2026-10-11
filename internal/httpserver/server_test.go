package httpserver_test

import (
	"bytes"
	"io"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"bakku.dev/bookist/internal/httpserver"
)

// ── Routing ───────────────────────────────────────────────────────────────────

func TestServerComposesAPIAndWebRoutes(t *testing.T) {
	server, err := httpserver.New(nil, nil, nil, nil, nil, slog.New(slog.NewTextHandler(io.Discard, nil)))
	if err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name   string
		method string
		path   string
		status int
	}{
		{name: "API", method: http.MethodPatch, path: "/api/books", status: http.StatusMethodNotAllowed},
		{name: "web", method: http.MethodGet, path: "/static/app.css", status: http.StatusOK},
		{name: "unknown API path", method: http.MethodGet, path: "/api/nope", status: http.StatusNotFound},
		{name: "unknown web path", method: http.MethodGet, path: "/does-not-exist", status: http.StatusNotFound},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			req := httptest.NewRequest(test.method, test.path, nil)
			resp := httptest.NewRecorder()

			server.Handler().ServeHTTP(resp, req)

			if resp.Code != test.status {
				t.Fatalf("expected status %d, got %d", test.status, resp.Code)
			}
		})
	}
}

// ── Request Logging ───────────────────────────────────────────────────────────

func TestServerLogsEachRequest(t *testing.T) {
	logs := &bytes.Buffer{}
	server, err := httpserver.New(nil, nil, nil, nil, nil, slog.New(slog.NewTextHandler(logs, nil)))
	if err != nil {
		t.Fatal(err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/nope", nil)
	resp := httptest.NewRecorder()
	server.Handler().ServeHTTP(resp, req)

	logged := logs.String()
	for _, expected := range []string{"level=INFO", "method=GET", "path=/api/nope", "status=404", "duration="} {
		if !strings.Contains(logged, expected) {
			t.Errorf("expected log to contain %q, got %s", expected, logged)
		}
	}
}
