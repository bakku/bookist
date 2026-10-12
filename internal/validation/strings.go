package validation

import "strings"

// TrimToNil trims surrounding whitespace and returns nil for a nil, empty or
// whitespace-only value, so blank optional strings are stored as NULL.
func TrimToNil(value *string) *string {
	if value == nil {
		return nil
	}
	trimmed := strings.TrimSpace(*value)
	if trimmed == "" {
		return nil
	}
	return &trimmed
}
