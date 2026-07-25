package reads

import "errors"

var ErrBookNotFound = errors.New("book not found")
var ErrReadNotFound = errors.New("read not found")
var ErrInvalidStartedAt = errors.New("started_at must be a date in YYYY-MM-DD format")
var ErrInvalidFinishedAt = errors.New("finished_at must be a date in YYYY-MM-DD format")
var ErrInvalidAbandonedAt = errors.New("abandoned_at must be a date in YYYY-MM-DD format")
var ErrConflictingTerminalDates = errors.New("finished_at and abandoned_at must not both be set")
var ErrFinishedBeforeStarted = errors.New("finished_at must not be before started_at")
var ErrAbandonedBeforeStarted = errors.New("abandoned_at must not be before started_at")
var ErrInvalidRating = errors.New("rating must be between 1 and 5 in increments of 0.5")
var ErrNoFieldsToUpdate = errors.New("no fields to update")
var ErrBlankOptionalString = errors.New("optional string fields must not be blank; use null to clear")
