package cli

import (
	"errors"
	"fmt"
	"strconv"
)

func parseIDReference(value string) (int64, bool, error) {
	id, err := strconv.ParseInt(value, 10, 64)
	if err == nil {
		if id <= 0 {
			return 0, true, fmt.Errorf("invalid ID %q", value)
		}
		return id, true, nil
	}
	if errors.Is(err, strconv.ErrRange) {
		return 0, true, fmt.Errorf("invalid ID %q", value)
	}
	return 0, false, nil
}
