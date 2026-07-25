package books

import "errors"

var ErrTitleRequired = errors.New("title is required")
var ErrAuthorNotFound = errors.New("author not found")
var ErrBookNotFound = errors.New("book not found")
var ErrInvalidFormat = errors.New("format must be one of: hardback, paperback, epub")
var ErrInvalidPurchasedAt = errors.New("purchased_at must be a date in YYYY-MM-DD format")
var ErrInvalidPages = errors.New("pages must be at least 1")
var ErrInvalidCondition = errors.New("condition must be one of: new, very_good, good, acceptable, poor")
var ErrInvalidSeriesPosition = errors.New("series_position must be greater than 0")
var ErrInvalidPublishedYear = errors.New("published_year must be at least 1")
var ErrInvalidPublishedMonth = errors.New("published_month must be between 1 and 12 and requires published_year")
var ErrInvalidPublishedDay = errors.New("published_day must form a valid date and requires published_year and published_month")
