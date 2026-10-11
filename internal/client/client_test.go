package client_test

import (
	"context"
	"database/sql"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"bakku.dev/bookist/internal/authors"
	"bakku.dev/bookist/internal/books"
	"bakku.dev/bookist/internal/client"
	"bakku.dev/bookist/internal/covers"
	"bakku.dev/bookist/internal/httpserver/api"
	"bakku.dev/bookist/internal/lists"
	"bakku.dev/bookist/internal/reads"
	"bakku.dev/bookist/internal/testsupport"
)

// newAPIHandler returns the real API handler backed by a migrated temporary database.
func newAPIHandler(t *testing.T) (http.Handler, *sql.DB) {
	t.Helper()

	db := testsupport.OpenMigratedDB(t)

	authorRepo := authors.NewSQLiteRepository(db)
	coverStore, err := covers.NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	mux := http.NewServeMux()
	api.New(
		books.NewService(books.NewSQLiteRepository(db), authorRepo, coverStore),
		authors.NewService(authorRepo),
		lists.NewService(lists.NewSQLiteRepository(db)),
		reads.NewService(reads.NewSQLiteRepository(db)),
		slog.New(slog.DiscardHandler),
	).RegisterRoutes(mux)

	return mux, db
}

// newTestClient returns a client connected to the real API handler over HTTP.
func newTestClient(t *testing.T) (*client.Client, *sql.DB) {
	t.Helper()

	handler, db := newAPIHandler(t)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	c, err := client.New(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	return c, db
}

// newDeadlineClient returns a client connected to the real API handler whose
// remaining function reports the time left on the last request's deadline.
func newDeadlineClient(t *testing.T) (*client.Client, *sql.DB, func() (time.Duration, bool)) {
	t.Helper()

	handler, db := newAPIHandler(t)
	server := httptest.NewServer(handler)
	t.Cleanup(server.Close)

	var deadline time.Time
	var hasDeadline bool
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		deadline, hasDeadline = r.Context().Deadline()
		return http.DefaultTransport.RoundTrip(r)
	})}

	c, err := client.New(server.URL, client.WithHTTPClient(httpClient))
	if err != nil {
		t.Fatal(err)
	}

	return c, db, func() (time.Duration, bool) { return time.Until(deadline), hasDeadline }
}

// assertDeadline checks that a request had a deadline in (lower, upper] from now.
func assertDeadline(t *testing.T, remaining func() (time.Duration, bool), lower, upper time.Duration) {
	t.Helper()

	left, hasDeadline := remaining()
	if !hasDeadline {
		t.Fatal("expected the request to have a deadline")
	}
	if left <= lower || left > upper {
		t.Fatalf("expected deadline in (%s, %s], got %s", lower, upper, left)
	}
}

// assertClientError checks that err is a *client.Error with the given details.
func assertClientError(t *testing.T, err error, status int, code, field string) *client.Error {
	t.Helper()

	var apiErr *client.Error
	if !errors.As(err, &apiErr) {
		t.Fatalf("expected *client.Error, got %#v", err)
	}
	if apiErr.Status != status || apiErr.Code != code || apiErr.Field != field || apiErr.Message == "" {
		t.Fatalf("expected status %d code %q field %q with a message, got %#v", status, code, field, apiErr)
	}

	return apiErr
}

// ── Construction ──────────────────────────────────────────────────────────────

func TestNewRejectsInvalidBaseURL(t *testing.T) {
	for _, baseURL := range []string{"localhost:8080", "bookist", "ftp://localhost", "http://", "http://host:bad"} {
		t.Run(baseURL, func(t *testing.T) {
			if _, err := client.New(baseURL); err == nil {
				t.Fatal("expected invalid base URL error")
			}
		})
	}
}

func TestClientKeepsBaseURLPathPrefix(t *testing.T) {
	handler, db := newAPIHandler(t)
	testsupport.InsertAuthorRow(t, db, "Frank Herbert")

	mux := http.NewServeMux()
	mux.Handle("/bookist/", http.StripPrefix("/bookist", handler))
	server := httptest.NewServer(mux)
	defer server.Close()

	for _, baseURL := range []string{server.URL + "/bookist", server.URL + "/bookist/"} {
		c, err := client.New(baseURL)
		if err != nil {
			t.Fatal(err)
		}

		listed, err := c.Authors.List(context.Background(), "")
		if err != nil {
			t.Fatalf("%s: %v", baseURL, err)
		}
		if len(listed) != 1 || listed[0].Name != "Frank Herbert" {
			t.Fatalf("%s: unexpected authors %#v", baseURL, listed)
		}
	}
}

func TestClientUsesProvidedHTTPClient(t *testing.T) {
	handler, _ := newAPIHandler(t)
	server := httptest.NewServer(handler)
	defer server.Close()

	var used bool
	httpClient := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		used = true
		return http.DefaultTransport.RoundTrip(r)
	})}

	c, err := client.New(server.URL, client.WithHTTPClient(httpClient))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := c.Books.List(context.Background(), ""); err != nil {
		t.Fatal(err)
	}
	if !used {
		t.Fatal("expected the provided HTTP client to be used")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

// ── Errors ────────────────────────────────────────────────────────────────────

func TestClientExposesStructuredBadRequestError(t *testing.T) {
	c, db := newTestClient(t)

	format := books.Format("foo")
	_, err := c.Books.Create(context.Background(), books.CreateBookRequest{Title: "Dune", Format: &format})

	apiErr := assertClientError(t, err, http.StatusBadRequest, "invalid_format", "format")
	if apiErr.Message != "format must be one of: hardback, paperback, epub" {
		t.Fatalf("unexpected message %q", apiErr.Message)
	}
	if got, want := err.Error(), "server returned 400 Bad Request: format must be one of: hardback, paperback, epub"; got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}
	testsupport.AssertSQLCount(t, db, 0, `SELECT COUNT(*) FROM books`)
}

func TestClientExposesNotFoundAndConflictErrors(t *testing.T) {
	c, db := newTestClient(t)
	testsupport.InsertListRow(t, db, "Nightstand")

	err := c.Books.Delete(context.Background(), 999)
	assertClientError(t, err, http.StatusNotFound, "book_not_found", "")

	_, err = c.Lists.Create(context.Background(), lists.CreateListRequest{Name: "Nightstand"})
	assertClientError(t, err, http.StatusConflict, "list_name_conflict", "name")
}

func TestClientReportsErrorFromPlainTextBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		http.Error(w, "upstream unavailable", http.StatusBadGateway)
	}))
	defer server.Close()

	c, err := client.New(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.Books.List(context.Background(), "")

	apiErr := assertClientError(t, err, http.StatusBadGateway, "", "")
	if apiErr.Message != "upstream unavailable" {
		t.Fatalf("unexpected message %q", apiErr.Message)
	}
	if got, want := err.Error(), "server returned 502 Bad Gateway: upstream unavailable"; got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}
}

func TestClientReportsErrorWithoutBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	c, err := client.New(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	err = c.Books.Delete(context.Background(), 1)

	var apiErr *client.Error
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusInternalServerError || apiErr.Message != "" {
		t.Fatalf("unexpected error %#v", err)
	}
	if got, want := err.Error(), "server returned 500 Internal Server Error"; got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}
}

func TestClientRejectsUnexpectedSuccessStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	c, err := client.New(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	err = c.Books.Delete(context.Background(), 1)

	var apiErr *client.Error
	if !errors.As(err, &apiErr) || apiErr.Status != http.StatusOK {
		t.Fatalf("expected unexpected status error, got %#v", err)
	}
	if got, want := err.Error(), "server returned 200 OK"; got != want {
		t.Fatalf("expected error %q, got %q", want, got)
	}
}

func TestClientReportsMalformedResponseBody(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("not json"))
	}))
	defer server.Close()

	c, err := client.New(server.URL)
	if err != nil {
		t.Fatal(err)
	}

	_, err = c.Books.List(context.Background(), "")
	if err == nil || !strings.Contains(err.Error(), "decode response") {
		t.Fatalf("expected decode error, got %v", err)
	}
}

func TestClientHonorsContextCancellation(t *testing.T) {
	c, _ := newTestClient(t)

	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	if _, err := c.Books.List(ctx, ""); !errors.Is(err, context.Canceled) {
		t.Fatalf("expected context canceled, got %v", err)
	}
}
