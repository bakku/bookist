package httpserver_test

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"bakku.dev/bookist/internal/httpserver"
)

func TestServerComposesAPIAndWebRoutes(t *testing.T) {
	server, err := httpserver.New(nil, nil, nil, nil, nil)
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
