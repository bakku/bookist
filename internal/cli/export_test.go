package cli

import (
	"net/http"
	"testing"
)

// SetLookupEnv replaces the environment lookup used for flag defaults until
// the test ends.
func SetLookupEnv(t *testing.T, lookup func(string) (string, bool)) {
	t.Helper()

	previous := lookupEnv
	lookupEnv = lookup
	t.Cleanup(func() { lookupEnv = previous })
}

// SetCoverTransport replaces the transport used to download covers until the
// test ends.
func SetCoverTransport(t *testing.T, transport http.RoundTripper) {
	t.Helper()

	previous := coverTransport
	coverTransport = transport
	t.Cleanup(func() { coverTransport = previous })
}
