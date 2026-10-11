package cli

import (
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

// supplied turns an optional flag value into an update field: absent when the
// flag was not given, otherwise set to the flag's value.
func supplied[T any](value *T) optional.Value[T] {
	if value == nil {
		return optional.Value[T]{}
	}
	return optional.Set(*value)
}
