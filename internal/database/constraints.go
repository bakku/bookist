package database

import (
	"errors"
	"strings"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

// IsUniqueViolation reports whether err is a SQLite UNIQUE constraint failure.
func IsUniqueViolation(err error) bool {
	return hasCode(err, sqlite3.SQLITE_CONSTRAINT_UNIQUE)
}

// IsForeignKeyViolation reports whether err is a SQLite FOREIGN KEY constraint failure.
func IsForeignKeyViolation(err error) bool {
	return hasCode(err, sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY)
}

// IsCheckViolation reports whether err is a failure of the named SQLite CHECK constraint.
func IsCheckViolation(err error, constraintName string) bool {
	var sqliteErr *sqlite.Error
	if !errors.As(err, &sqliteErr) || sqliteErr.Code() != sqlite3.SQLITE_CONSTRAINT_CHECK {
		return false
	}

	// SQLite reports named CHECK failures as "CHECK constraint failed: <name>".
	_, detail, found := strings.Cut(sqliteErr.Error(), "CHECK constraint failed: ")
	if !found {
		return false
	}
	name, _, _ := strings.Cut(detail, " ")
	return name == constraintName
}

func hasCode(err error, code int) bool {
	var sqliteErr *sqlite.Error
	return errors.As(err, &sqliteErr) && sqliteErr.Code() == code
}
