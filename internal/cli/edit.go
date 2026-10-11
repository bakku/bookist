package cli

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
)

const partialUpdateHelp = "Omitted fields are unchanged; supplied values replace fields; --clear FIELD clears nullable fields. Blank strings are rejected rather than cleared."

// clearTarget is an optional update field that --clear can reset to null.
type clearTarget interface {
	IsZero() bool
	Clear()
}

func clearFlagUsage[T any](clearable map[string]T) string {
	names := make([]string, 0, len(clearable))
	for name := range clearable {
		names = append(names, name)
	}
	sort.Strings(names)
	return "Clear a nullable field (repeatable): " + strings.Join(names, ", ")
}

// applyClears clears the fields named by --clear. A field that was also
// supplied a value cannot be cleared.
func applyClears(clearable map[string]clearTarget, clears []string) error {
	cleared := make(map[string]bool)
	for _, name := range clears {
		target, ok := clearable[name]
		if !ok {
			return fmt.Errorf("field %q cannot be cleared", name)
		}
		if !target.IsZero() && !cleared[name] {
			return fmt.Errorf("field %q cannot be set and cleared", name)
		}
		target.Clear()
		cleared[name] = true
	}
	return nil
}

// isEmptyUpdate reports whether an update request would change nothing.
func isEmptyUpdate(request any) bool {
	encoded, err := json.Marshal(request)
	return err == nil && string(encoded) == "{}"
}
