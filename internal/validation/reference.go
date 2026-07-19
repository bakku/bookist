package validation

import (
	"time"
)

func IsCalendarDate(value string) bool {
	date, err := time.Parse(time.DateOnly, value)
	return err == nil && date.Year() >= 1
}
