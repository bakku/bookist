package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"text/tabwriter"
)

type helpCommand struct {
	name        string
	description string
}

type commandHelp struct {
	name        string
	usage       string
	description string
	details     []string
	examples    []string
	commands    []helpCommand
}

func printCommandHelp(w io.Writer, help commandHelp, flags *flag.FlagSet) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)

	_, _ = fmt.Fprintf(tw, "NAME:\n   %s - %s\n\n", help.name, help.description)
	_, _ = fmt.Fprintf(tw, "USAGE:\n   %s\n", help.usage)

	if len(help.details) > 0 {
		_, _ = fmt.Fprintln(tw, "\nDESCRIPTION:")
		for _, detail := range help.details {
			_, _ = fmt.Fprintf(tw, "   %s\n", detail)
		}
	}

	if len(help.examples) > 0 {
		_, _ = fmt.Fprintln(tw, "\nEXAMPLES:")
		for _, example := range help.examples {
			_, _ = fmt.Fprintf(tw, "   %s\n", example)
		}
	}

	if len(help.commands) > 0 {
		_, _ = fmt.Fprintln(tw, "\nCOMMANDS:")

		for _, command := range help.commands {
			_, _ = fmt.Fprintf(tw, "   %s\t%s\n", command.name, command.description)
		}
	}

	_, _ = fmt.Fprintln(tw, "\nOPTIONS:")

	if flags != nil {
		flags.VisitAll(func(f *flag.Flag) {
			if f.Name == "o" {
				return
			}

			option := "--" + f.Name
			if f.Name == "output" {
				option += ", -o"
			}
			option += flagPlaceholder(f)
			description := f.Usage

			if showDefault(f.DefValue) {
				description += " (default: " + f.DefValue + ")"
			}

			_, _ = fmt.Fprintf(tw, "   %s\t%s\n", option, description)
		})
	}
	_, _ = fmt.Fprintln(tw, "   --help, -h\tShow help")

	_ = tw.Flush()
}

const (
	mutationOutputHelp = "With --output json, prints the resulting record as JSON instead of text."
	removedOutputHelp  = `With --output json, prints {"id": N, "removed": true} instead of text.`

	membershipAddOutputHelp    = `With --output json, prints {"list_id": N, "book_id": N, "added": true} instead of text.`
	membershipRemoveOutputHelp = `With --output json, prints {"list_id": N, "book_id": N, "removed": true} instead of text.`
)

// parseFlags parses command arguments. Flags may appear before, between and
// after positional arguments.
func parseFlags(flags *flag.FlagSet, args []string, stdout, stderr io.Writer, help commandHelp) (bool, int) {
	flags.SetOutput(stderr)
	flags.Usage = func() {}

	err := flags.Parse(interspersedArgs(flags, args))
	if errors.Is(err, flag.ErrHelp) {
		printCommandHelp(stdout, help, flags)
		return false, 0
	}
	if err != nil {
		_, _ = fmt.Fprintln(stderr)
		printCommandHelp(stderr, help, flags)
		return false, 2
	}

	return true, 0
}

func flagPlaceholder(f *flag.Flag) string {
	switch f.Value.(type) {
	case *optionalStringFlag, *stringSliceFlag:
		return " string"
	case *optionalIntFlag:
		return " int"
	case *optionalFloatFlag:
		return " float"
	}

	getter, ok := f.Value.(flag.Getter)
	if !ok {
		return " value"
	}

	switch getter.Get().(type) {
	case bool:
		return ""
	case int:
		return " int"
	case string:
		return " string"
	default:
		return " value"
	}
}

func showDefault(value string) bool {
	return value != "" && value != "0" && value != "false"
}

// rejectArgs reports stray positional arguments for a command that takes none.
// It returns true when there are none.
func rejectArgs(flags *flag.FlagSet, command string, stderr io.Writer, help commandHelp) bool {
	if flags.NArg() == 0 {
		return true
	}

	_, _ = fmt.Fprintf(stderr, "Error: %s does not accept positional arguments\n", command)
	_, _ = fmt.Fprintln(stderr)
	printCommandHelp(stderr, help, flags)

	return false
}
