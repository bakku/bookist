package cli

import "testing"

// SetLookupEnv replaces the environment lookup used for flag defaults until
// the test ends.
func SetLookupEnv(t *testing.T, lookup func(string) (string, bool)) {
	t.Helper()

	previous := lookupEnv
	lookupEnv = lookup
	t.Cleanup(func() { lookupEnv = previous })
}
