package database

import (
	"errors"
	"strings"

	"modernc.org/sqlite"
	sqlite3 "modernc.org/sqlite/lib"
)

const checkErrorPrefix = "CHECK constraint failed: "

// IsUniqueViolation checks the extended code, including wrapped driver errors.
func IsUniqueViolation(err error) bool {
	return hasConstraintCode(err, sqlite3.SQLITE_CONSTRAINT_UNIQUE)
}

// IsForeignKeyViolation identifies missing referenced rows.
func IsForeignKeyViolation(err error) bool {
	return hasConstraintCode(err, sqlite3.SQLITE_CONSTRAINT_FOREIGNKEY)
}

// IsCheckViolation matches a named check only after verifying its error code.
func IsCheckViolation(err error, constraintName string) bool {
	var sqliteErr *sqlite.Error
	if constraintName == "" || !errors.As(err, &sqliteErr) || sqliteErr.Code() != sqlite3.SQLITE_CONSTRAINT_CHECK {
		return false
	}

	// SQLite exposes check names in the message; the delimiter prevents prefix matches.
	return strings.Contains(sqliteErr.Error(), checkErrorPrefix+constraintName+" (")
}

func hasConstraintCode(err error, code int) bool {
	var sqliteErr *sqlite.Error
	return errors.As(err, &sqliteErr) && sqliteErr.Code() == code
}
