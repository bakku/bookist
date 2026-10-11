package database

import (
	"database/sql/driver"
	"strings"
	"sync"
	"unicode"

	"golang.org/x/text/cases"
	"golang.org/x/text/runes"
	"golang.org/x/text/transform"
	"golang.org/x/text/unicode/norm"

	"modernc.org/sqlite"
)

// foldFunction is the SQLite scalar function that applies Fold to a column, so
// searches can compare folded text with instr(bookist_fold(column), ?).
const foldFunction = "bookist_fold"

var registerFoldOnce sync.Once

// Fold makes text comparable regardless of case and accents: it removes
// combining marks after canonical decomposition and then applies Unicode case
// folding, so "Über" and "ÜBER" both become "uber" and "Straße" becomes
// "strasse". Callers fold search input with Fold and let SQLite fold the
// stored column with bookist_fold.
func Fold(s string) string {
	// The transformers keep state, so build a fresh chain for every call.
	folded, _, err := transform.String(transform.Chain(
		norm.NFD,
		runes.Remove(runes.In(unicode.Mn)),
		cases.Fold(),
		norm.NFC,
	), s)
	if err != nil {
		return strings.ToLower(s)
	}
	return folded
}

// registerFoldFunction makes bookist_fold available to every connection opened
// afterwards. SQLite then folds NULL to NULL, like any other scalar function.
func registerFoldFunction() {
	registerFoldOnce.Do(func() {
		err := sqlite.RegisterDeterministicScalarFunction(foldFunction, 1,
			func(_ *sqlite.FunctionContext, args []driver.Value) (driver.Value, error) {
				switch value := args[0].(type) {
				case string:
					return Fold(value), nil
				case []byte:
					return Fold(string(value)), nil
				default:
					return value, nil
				}
			})
		if err != nil {
			panic("register " + foldFunction + ": " + err.Error())
		}
	})
}
