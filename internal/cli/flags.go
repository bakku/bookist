package cli

import (
	"flag"
	"strconv"
	"strings"

	"bakku.dev/bookist/internal/optional"
)

type stringSliceFlag []string

func (s *stringSliceFlag) String() string {
	return strings.Join(*s, ",")
}

func (s *stringSliceFlag) Set(value string) error {
	*s = append(*s, value)
	return nil
}

type optionalStringFlag struct {
	value *string
}

func (f *optionalStringFlag) String() string {
	if f.value == nil {
		return ""
	}
	return *f.value
}

func (f *optionalStringFlag) Set(value string) error {
	f.value = &value
	return nil
}

type optionalIntFlag struct {
	value *int
}

type optionalFloatFlag struct {
	value *float64
}

func (f *optionalFloatFlag) String() string {
	if f.value == nil {
		return ""
	}
	return strconv.FormatFloat(*f.value, 'f', -1, 64)
}

func (f *optionalFloatFlag) Set(value string) error {
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return err
	}
	f.value = &parsed
	return nil
}

func (f *optionalIntFlag) String() string {
	if f.value == nil {
		return ""
	}
	return strconv.Itoa(*f.value)
}

func (f *optionalIntFlag) Set(value string) error {
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return err
	}
	f.value = &parsed
	return nil
}

// interspersedArgs reorders args so that flags come before positional
// arguments, which lets flags follow positionals: flag.FlagSet.Parse stops at
// the first positional. A flag missing its value is left last so that Parse
// reports it.
func interspersedArgs(flags *flag.FlagSet, args []string) []string {
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
		name, hasValue := flagName(arg)
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
		return options
	}
	return append(append(options, "--"), operands...)
}

func flagName(arg string) (string, bool) {
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

// supplied turns an optional flag value into an update field: absent when the
// flag was not given, otherwise set to the flag's value.
func supplied[T any](value *T) optional.Value[T] {
	if value == nil {
		return optional.Value[T]{}
	}
	return optional.Set(*value)
}
