// Package client is a typed client for the Bookist HTTP JSON API.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultTimeout   = 10 * time.Second
	maxErrorBodySize = 4 << 10

	// CoverTimeout is the time allowed for cover downloads and for requests
	// that carry cover bytes, which can be much larger than other requests.
	CoverTimeout = 2 * time.Minute
)

// Client talks to a Bookist server. Resource operations are grouped by
// resource, for example c.Books.List and c.Lists.AddBook.
type Client struct {
	baseURL    *url.URL
	httpClient *http.Client

	Books   *BooksService
	Authors *AuthorsService
	Lists   *ListsService
	Reads   *ReadsService
}

// Option customizes a Client.
type Option func(*Client)

// WithHTTPClient replaces the default HTTP client. Requests are limited by a
// context deadline of 10 seconds, or CoverTimeout for requests that carry a
// cover, in addition to any Timeout set on httpClient.
func WithHTTPClient(httpClient *http.Client) Option {
	return func(c *Client) {
		c.httpClient = httpClient
	}
}

// New returns a Client for the server at baseURL, for example
// "http://localhost:8080". The URL must be an absolute http or https URL.
func New(baseURL string, options ...Option) (*Client, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	if (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, fmt.Errorf("%q is not an absolute http or https URL", baseURL)
	}

	c := &Client{
		baseURL:    parsed,
		httpClient: &http.Client{},
	}
	for _, option := range options {
		option(c)
	}

	c.Books = &BooksService{client: c}
	c.Authors = &AuthorsService{client: c}
	c.Lists = &ListsService{client: c}
	c.Reads = &ReadsService{client: c}

	return c, nil
}

// Error is returned when the server answers with an unexpected status. Code,
// Message and Field come from the structured JSON error body when the server
// sent one; otherwise Message holds the raw response body.
type Error struct {
	Status  int
	Code    string
	Message string
	Field   string
}

func (e *Error) Error() string {
	status := fmt.Sprintf("%d %s", e.Status, http.StatusText(e.Status))
	if e.Message == "" {
		return "server returned " + status
	}
	return "server returned " + status + ": " + e.Message
}

// endpoint builds the URL for path segments below the base URL. A blank query
// is not sent.
func (c *Client) endpoint(query string, segments ...string) string {
	endpoint := c.baseURL.JoinPath(segments...)
	if strings.TrimSpace(query) != "" {
		endpoint.RawQuery = url.Values{"q": {query}}.Encode()
	}
	return endpoint.String()
}

// do sends a request with an optional JSON body and decodes a JSON response
// into out when out is not nil. Any status other than wantStatus is an *Error.
func (c *Client) do(ctx context.Context, method, endpoint string, body, out any, wantStatus int) error {
	return c.doWithTimeout(ctx, defaultTimeout, method, endpoint, body, out, wantStatus)
}

// doWithTimeout is do with a custom deadline covering the whole exchange,
// including reading the response body.
func (c *Client) doWithTimeout(ctx context.Context, timeout time.Duration, method, endpoint string, body, out any, wantStatus int) error {
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	var reader io.Reader
	if body != nil {
		encoded, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode request: %w", err)
		}
		reader = bytes.NewReader(encoded)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint, reader)
	if err != nil {
		return err
	}
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != wantStatus {
		return newError(resp)
	}
	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("decode response: %w", err)
	}
	return nil
}

func newError(resp *http.Response) *Error {
	apiErr := &Error{Status: resp.StatusCode}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxErrorBodySize))
	if err != nil {
		return apiErr
	}

	var structured struct {
		Error struct {
			Code    string `json:"code"`
			Message string `json:"message"`
			Field   string `json:"field"`
		} `json:"error"`
	}
	if json.Unmarshal(body, &structured) == nil && structured.Error.Message != "" {
		apiErr.Code = structured.Error.Code
		apiErr.Message = structured.Error.Message
		apiErr.Field = structured.Error.Field
		return apiErr
	}

	apiErr.Message = strings.TrimSpace(string(body))
	return apiErr
}
