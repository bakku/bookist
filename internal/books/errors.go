package books

import "bakku.dev/bookist/internal/apperr"

var ErrTitleRequired = apperr.Invalid("title_required", "title is required", "title")
var ErrAuthorNotFound = apperr.Invalid("author_not_found", "author not found", "author_ids")
var ErrInvalidFormat = apperr.Invalid("invalid_format", "format must be one of: hardback, paperback, epub", "format")
var ErrInvalidPurchasedAt = apperr.Invalid("invalid_purchased_at", "purchased_at must be a date in YYYY-MM-DD format", "purchased_at")
var ErrInvalidPages = apperr.Invalid("invalid_pages", "pages must be at least 1", "pages")
var ErrInvalidCondition = apperr.Invalid("invalid_condition", "condition must be one of: new, very_good, good, acceptable, poor", "condition")
var ErrInvalidSeriesPosition = apperr.Invalid("invalid_series_position", "series_position must be greater than 0", "series_position")
var ErrInvalidPublishedYear = apperr.Invalid("invalid_published_year", "published_year must be at least 1", "published_year")
var ErrInvalidPublishedMonth = apperr.Invalid("invalid_published_month", "published_month must be between 1 and 12 and requires published_year", "published_month")
var ErrInvalidPublishedDay = apperr.Invalid("invalid_published_day", "published_day must form a valid date and requires published_year and published_month", "published_day")
var ErrNoFieldsToUpdate = apperr.Invalid("no_fields_to_update", "no fields to update", "")
