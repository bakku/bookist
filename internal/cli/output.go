package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

type outputFormat string

const (
	outputFormatTSV    outputFormat = "tsv"
	outputFormatPretty outputFormat = "pretty"
	outputFormatJSON   outputFormat = "json"
)

func parseOutputFormat(value string) (outputFormat, error) {
	format := outputFormat(value)
	switch format {
	case outputFormatTSV, outputFormatPretty, outputFormatJSON:
		return format, nil
	default:
		return "", fmt.Errorf("unsupported output format %q: must be one of tsv, pretty, json", value)
	}
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
