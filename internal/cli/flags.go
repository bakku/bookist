package cli

import (
	"strconv"
	"strings"
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
