package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"sort"
	"strings"
)

const partialUpdateHelp = "Omitted fields are unchanged; supplied values replace fields; --clear FIELD clears nullable fields. Blank strings are rejected rather than cleared."

func parseEditFlags(flags *flag.FlagSet, args []string, stdout, stderr io.Writer, help commandHelp) (bool, int) {
	options := make([]string, 0, len(args))
	operands := make([]string, 0, 1)
	missingValue := false

	for i := 0; i < len(args); i++ {
		arg := args[i]
		if arg == "--" {
			operands = append(operands, args[i+1:]...)
			break
		}
		if arg == "-" || !strings.HasPrefix(arg, "-") {
			operands = append(operands, arg)
			continue
		}

		options = append(options, arg)
		name, hasValue := editFlagName(arg)
		if hasValue || name == "h" || name == "help" {
			continue
		}
		registered := flags.Lookup(name)
		if registered == nil || isBooleanFlag(registered) {
			continue
		}
		if i+1 < len(args) {
			i++
			options = append(options, args[i])
		} else {
			missingValue = true
		}
	}

	if missingValue {
		return parseFlags(flags, options, stdout, stderr, help)
	}
	reordered := append(options, "--")
	reordered = append(reordered, operands...)
	return parseFlags(flags, reordered, stdout, stderr, help)
}

func editFlagName(arg string) (string, bool) {
	name := strings.TrimLeft(arg, "-")
	name, _, hasValue := strings.Cut(name, "=")
	return name, hasValue
}

func isBooleanFlag(flag *flag.Flag) bool {
	type boolFlag interface {
		IsBoolFlag() bool
	}
	value, ok := flag.Value.(boolFlag)
	return ok && value.IsBoolFlag()
}

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
