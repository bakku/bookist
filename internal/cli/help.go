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
	commands    []helpCommand
}

func printCommandHelp(w io.Writer, help commandHelp, flags *flag.FlagSet) {
	tw := tabwriter.NewWriter(w, 0, 4, 2, ' ', 0)
	_, _ = fmt.Fprintf(tw, "NAME:\n   %s - %s\n\n", help.name, help.description)
	_, _ = fmt.Fprintf(tw, "USAGE:\n   %s\n", help.usage)

	if len(help.commands) > 0 {
		_, _ = fmt.Fprintln(tw, "\nCOMMANDS:")
		for _, command := range help.commands {
			_, _ = fmt.Fprintf(tw, "   %s\t%s\n", command.name, command.description)
		}
	}

	_, _ = fmt.Fprintln(tw, "\nOPTIONS:")
	if flags != nil {
		flags.VisitAll(func(f *flag.Flag) {
			option := "--" + f.Name + flagPlaceholder(f)
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

func parseFlags(flags *flag.FlagSet, args []string, stdout, stderr io.Writer, help commandHelp) (bool, int) {
	flags.SetOutput(stderr)
	flags.Usage = func() {}

	err := flags.Parse(args)
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
