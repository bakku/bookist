// Package domain holds errors shared by several domain packages, so each
// concept has a single sentinel for errors.Is.
package domain

import "bakku.dev/bookist/internal/apperr"

var ErrBookNotFound = apperr.NotFound("book_not_found", "book not found")
var ErrListNotFound = apperr.NotFound("list_not_found", "list not found")
