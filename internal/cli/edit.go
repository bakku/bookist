package cli

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strings"
	"time"
)

const maxCLIErrorBodySize = 4 << 10

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

func clearFlagUsage(clearable map[string]string) string {
	names := make([]string, 0, len(clearable))
	for name := range clearable {
		names = append(names, name)
	}
	sort.Strings(names)
	return "Clear a nullable field (repeatable): " + strings.Join(names, ", ")
}

func validateClears(changes map[string]any, clears []string, clearable map[string]string) error {
	cleared := make(map[string]bool)
	for _, name := range clears {
		key, ok := clearable[name]
		if !ok {
			return fmt.Errorf("field %q cannot be cleared", name)
		}
		if _, exists := changes[key]; exists && !cleared[key] {
			return fmt.Errorf("field %q cannot be set and cleared", name)
		}
		changes[key] = nil
		cleared[key] = true
	}
	return nil
}

func patchEndpoint(endpoint string, changes map[string]any, output any) error {
	body, err := json.Marshal(changes)
	if err != nil {
		return err
	}
	req, err := http.NewRequest(http.MethodPatch, endpoint, bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	client := http.Client{Timeout: 10 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		body, readErr := io.ReadAll(io.LimitReader(resp.Body, maxCLIErrorBodySize))
		if readErr == nil {
			message := strings.TrimSpace(string(body))
			if message != "" {
				return fmt.Errorf("server returned %s: %s", resp.Status, message)
			}
		}
		return fmt.Errorf("server returned %s", resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(output); err != nil {
		return fmt.Errorf("decode response: %v", err)
	}
	return nil
}
