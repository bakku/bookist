package database

import (
	"database/sql"
	"fmt"
	"time"
)

// TimeLayout is the layout of every created_at and updated_at column. It is
// fixed-width (milliseconds, always UTC) so the text sorts chronologically;
// time.RFC3339Nano would drop trailing zeros and break that, and second
// precision would tie rows changed within the same second.
const TimeLayout = "2006-01-02T15:04:05.000Z07:00"

// Now returns the current time formatted for storage.
func Now() string {
	return FormatTime(time.Now())
}

// FormatTime formats t for storage in a timestamp column.
func FormatTime(t time.Time) string {
	return t.UTC().Format(TimeLayout)
}

// ParseTime parses a stored timestamp. time.RFC3339 accepts an optional
// fraction, so it also reads second-precision values.
func ParseTime(value string) (time.Time, error) {
	return time.Parse(time.RFC3339, value)
}

// Time returns a sql.Scanner that parses a stored timestamp into dest, so a
// row of timestamp columns can be scanned in the same call as the rest.
func Time(dest *time.Time) sql.Scanner {
	return timeScanner{dest: dest}
}

type timeScanner struct {
	dest *time.Time
}

func (s timeScanner) Scan(src any) error {
	var value string
	switch v := src.(type) {
	case string:
		value = v
	case []byte:
		value = string(v)
	default:
		return fmt.Errorf("scan timestamp: unsupported type %T", src)
	}

	parsed, err := ParseTime(value)
	if err != nil {
		return fmt.Errorf("parse timestamp: %w", err)
	}
	*s.dest = parsed

	return nil
}
