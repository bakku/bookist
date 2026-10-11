package web

import (
	"bytes"
	"html/template"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ── Render Errors ─────────────────────────────────────────────────────────────

func TestIndexLogsTemplateRenderFailure(t *testing.T) {
	logs := &bytes.Buffer{}
	server := &Server{
		templates: TemplateSet{BooksIndex: template.Must(template.New("layout").Parse(`{{.Missing}}`))},
		logger:    slog.New(slog.NewTextHandler(logs, nil)),
	}

	req := httptest.NewRequest(http.MethodGet, "/", nil)
	resp := httptest.NewRecorder()
	server.handleIndex(resp, req)

	if resp.Code != http.StatusInternalServerError {
		t.Fatalf("expected status %d, got %d", http.StatusInternalServerError, resp.Code)
	}

	logged := logs.String()
	for _, expected := range []string{"level=ERROR", "failed to render page", "method=GET", "path=/", "Missing"} {
		if !strings.Contains(logged, expected) {
			t.Errorf("expected log to contain %q, got %s", expected, logged)
		}
	}
}
