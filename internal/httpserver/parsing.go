package httpserver

import (
	"fmt"
	"strconv"
)

func parseID(value string) (int64, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err != nil || id <= 0 {
		return 0, fmt.Errorf("invalid ID")
	}
	return id, nil
}
