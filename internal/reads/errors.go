package reads

import "bakku.dev/bookist/internal/apperr"

var ErrReadNotFound = apperr.NotFound("read_not_found", "read not found")
var ErrInvalidStartedAt = apperr.Invalid("invalid_started_at", "started_at must be a date in YYYY-MM-DD format", "started_at")
var ErrInvalidFinishedAt = apperr.Invalid("invalid_finished_at", "finished_at must be a date in YYYY-MM-DD format", "finished_at")
var ErrInvalidAbandonedAt = apperr.Invalid("invalid_abandoned_at", "abandoned_at must be a date in YYYY-MM-DD format", "abandoned_at")
var ErrConflictingTerminalDates = apperr.Invalid("conflicting_terminal_dates", "finished_at and abandoned_at must not both be set", "")
var ErrFinishedBeforeStarted = apperr.Invalid("finished_before_started", "finished_at must not be before started_at", "finished_at")
var ErrAbandonedBeforeStarted = apperr.Invalid("abandoned_before_started", "abandoned_at must not be before started_at", "abandoned_at")
var ErrInvalidRating = apperr.Invalid("invalid_rating", "rating must be between 1 and 5 in increments of 0.5", "rating")
var ErrNoFieldsToUpdate = apperr.Invalid("no_fields_to_update", "no fields to update", "")
var ErrBlankOptionalString = apperr.Invalid("blank_optional_string", "optional string fields must not be blank; use null to clear", "")
