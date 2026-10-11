package cli

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strconv"
	"strings"
	"text/tabwriter"
	"time"
)

type outputFormat string

const (
	outputFormatTSV    outputFormat = "tsv"
	outputFormatPretty outputFormat = "pretty"
	outputFormatJSON   outputFormat = "json"
	outputFormatText   outputFormat = "text"

	outputFormatsHelp   = "tsv|pretty|json"
	mutationFormatsHelp = "text|json"
)

// outputFlag registers --output with its -o alias, both bound to one value.
func outputFlag(flags *flag.FlagSet, fallback outputFormat, formats string) *string {
	usage := "Output format (" + formats + ")"
	value := flags.String("output", string(fallback), usage)
	flags.StringVar(value, "o", string(fallback), usage)

	return value
}

func parseOutputFormat(value string) (outputFormat, error) {
	format := outputFormat(value)
	switch format {
	case outputFormatTSV, outputFormatPretty, outputFormatJSON:
		return format, nil
	default:
		return "", fmt.Errorf("unsupported output format %q: must be one of tsv, pretty, json", value)
	}
}

// parseMutationOutput parses the output format of add, edit and remove
// commands: text (the default) or json.
func parseMutationOutput(value string) (outputFormat, error) {
	format := outputFormat(value)
	switch format {
	case outputFormatText, outputFormatJSON:
		return format, nil
	default:
		return "", fmt.Errorf("unsupported output format %q: must be one of text, json", value)
	}
}

// writeMutationOutput reports the result of a mutation: text as a line, or
// value encoded as JSON.
func writeMutationOutput(w io.Writer, format outputFormat, value any, text string) error {
	if format == outputFormatJSON {
		return json.NewEncoder(w).Encode(value)
	}

	_, err := fmt.Fprintln(w, text)
	return err
}

// removedResult is the JSON output of the rm commands.
type removedResult struct {
	ID      int64 `json:"id"`
	Removed bool  `json:"removed"`
}

// addedToListResult is the JSON output of lists add-book.
type addedToListResult struct {
	ListID int64 `json:"list_id"`
	BookID int64 `json:"book_id"`
	Added  bool  `json:"added"`
}

// removedFromListResult is the JSON output of lists rm-book.
type removedFromListResult struct {
	ListID  int64 `json:"list_id"`
	BookID  int64 `json:"book_id"`
	Removed bool  `json:"removed"`
}

func writeListOutput(w io.Writer, format outputFormat, value any, headers []string, rows [][]string) error {
	switch format {
	case outputFormatTSV:
		return writeRows(w, rows)
	case outputFormatPretty:
		var formatted bytes.Buffer
		tw := tabwriter.NewWriter(&formatted, 0, 4, 2, ' ', 0)
		if err := writeRows(tw, append([][]string{headers}, rows...)); err != nil {
			return err
		}
		if err := tw.Flush(); err != nil {
			return err
		}

		lines := strings.Split(formatted.String(), "\n")
		for i := range lines {
			lines[i] = strings.TrimRight(lines[i], " ")
		}
		_, err := io.WriteString(w, strings.Join(lines, "\n"))
		return err
	case outputFormatJSON:
		return json.NewEncoder(w).Encode(value)
	default:
		return fmt.Errorf("unsupported output format %q", format)
	}
}

func writeRows(w io.Writer, rows [][]string) error {
	for _, row := range rows {
		if _, err := fmt.Fprintln(w, strings.Join(row, "\t")); err != nil {
			return err
		}
	}
	return nil
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func floatValue(value *float64) string {
	if value == nil {
		return ""
	}
	return strconv.FormatFloat(*value, 'f', -1, 64)
}

func intValue(value *int) string {
	if value == nil {
		return ""
	}
	return strconv.Itoa(*value)
}

func timeValue(value time.Time) string {
	return value.Format(time.RFC3339)
}

// showField is one labelled value of a single record in show output.
type showField struct {
	label string
	value string
}

// writeShowOutput writes a single record. TSV and pretty output list one
// label and value per line, JSON output encodes value.
func writeShowOutput(w io.Writer, format outputFormat, value any, fields []showField) error {
	rows := make([][]string, 0, len(fields))
	for _, field := range fields {
		rows = append(rows, []string{field.label, field.value})
	}

	switch format {
	case outputFormatTSV:
		return writeRows(w, rows)
	case outputFormatPretty:
		var formatted bytes.Buffer
		tw := tabwriter.NewWriter(&formatted, 0, 4, 2, ' ', 0)
		if err := writeRows(tw, rows); err != nil {
			return err
		}
		if err := tw.Flush(); err != nil {
			return err
		}

		lines := strings.Split(formatted.String(), "\n")
		for i := range lines {
			lines[i] = strings.TrimRight(lines[i], " ")
		}
		_, err := io.WriteString(w, strings.Join(lines, "\n"))
		return err
	case outputFormatJSON:
		return json.NewEncoder(w).Encode(value)
	default:
		return fmt.Errorf("unsupported output format %q", format)
	}
}
