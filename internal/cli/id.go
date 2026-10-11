package cli

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"bakku.dev/bookist/internal/client"
)

func parseIDReference(value string) (int64, bool, error) {
	id, err := strconv.ParseInt(value, 10, 64)

	if err == nil {
		if id <= 0 {
			return 0, true, fmt.Errorf("invalid ID %q", value)
		}

		return id, true, nil
	}

	if errors.Is(err, strconv.ErrRange) {
		return 0, true, fmt.Errorf("invalid ID %q", value)
	}

	return 0, false, nil
}

// explicitIDPrefix marks a book, author or list reference as an ID, for
// example id:12, so that it is never mistaken for a numeric title or name.
const explicitIDPrefix = "id:"

// referenceHelp describes the reference syntax for books, authors and lists.
const referenceHelp = "References are exact titles or names, or IDs. A number that is an ID and also a different record's title or name is rejected as ambiguous; write id:N (for example id:12) to always mean an ID."

// notFoundError reports a reference that matches no record.
type notFoundError struct {
	kind      string
	reference string
}

func (e *notFoundError) Error() string {
	return fmt.Sprintf("%s not found: %s", e.kind, e.reference)
}

// record is a stored book, author or list that a reference can match.
type record struct {
	ID   int64
	Name string
}

// referenceSource looks up one kind of record (book, author or list) by ID and
// by exact name for resolveReference.
type referenceSource struct {
	kind string
	// get returns the record with the ID, or a *client.Error with status 404.
	get func(ctx context.Context, id int64) (record, error)
	// find returns the records whose title or name equals name, ignoring case.
	find func(ctx context.Context, name string) ([]record, error)
}

// resolveReference turns a user-supplied reference into a record ID. id:N
// always means ID N. Any other reference is matched by exact title or name; a
// positive integer is also looked up as an ID, and matching different records
// by ID and by name is an ambiguity error instead of a guess.
func resolveReference(ctx context.Context, source referenceSource, value string) (int64, error) {
	value = strings.TrimSpace(value)

	if rest, ok := strings.CutPrefix(value, explicitIDPrefix); ok {
		id, isID, err := parseIDReference(rest)
		if err != nil || !isID {
			return 0, fmt.Errorf("invalid ID %q", value)
		}

		found, ok, err := lookupRecord(ctx, source, id)
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, &notFoundError{kind: source.kind, reference: value}
		}

		return found.ID, nil
	}

	var candidates []record
	var byID *record

	if id, isID, err := parseIDReference(value); err == nil && isID {
		found, ok, err := lookupRecord(ctx, source, id)
		if err != nil {
			return 0, err
		}
		if ok {
			byID = &found
			candidates = append(candidates, found)
		}
	}

	matches, err := source.find(ctx, value)
	if err != nil {
		return 0, fmt.Errorf("fetch %ss: %w", source.kind, err)
	}
	for _, match := range matches {
		if byID == nil || match.ID != byID.ID {
			candidates = append(candidates, match)
		}
	}

	switch {
	case len(candidates) == 0:
		return 0, &notFoundError{kind: source.kind, reference: value}

	case len(candidates) == 1:
		return candidates[0].ID, nil

	case byID == nil:
		return 0, fmt.Errorf("%s %q exists multiple times; pass %s %s ID instead", source.kind, value, indefiniteArticle(source.kind), source.kind)
	}

	described := make([]string, 0, len(candidates))
	suggested := make([]string, 0, len(candidates))
	for _, candidate := range candidates {
		described = append(described, fmt.Sprintf("%s %d (%q)", source.kind, candidate.ID, candidate.Name))
		suggested = append(suggested, fmt.Sprintf("%s%d", explicitIDPrefix, candidate.ID))
	}

	return 0, fmt.Errorf("ambiguous %s reference %q: matches %s; use %s",
		source.kind, value, joinWords(described, "and"), joinWords(suggested, "or"))
}

// lookupRecord fetches a record by ID, reporting a missing record as not ok
// rather than as an error.
func lookupRecord(ctx context.Context, source referenceSource, id int64) (record, bool, error) {
	found, err := source.get(ctx, id)

	var apiErr *client.Error
	if errors.As(err, &apiErr) && apiErr.Status == http.StatusNotFound {
		return record{}, false, nil
	}
	if err != nil {
		return record{}, false, fmt.Errorf("fetch %s: %w", source.kind, err)
	}

	return found, true, nil
}

// joinWords joins items as "a", "a and b" or "a, b and c".
func joinWords(items []string, conjunction string) string {
	if len(items) < 2 {
		return strings.Join(items, "")
	}

	return strings.Join(items[:len(items)-1], ", ") + " " + conjunction + " " + items[len(items)-1]
}

func indefiniteArticle(word string) string {
	if strings.ContainsAny(word[:1], "aeiou") {
		return "an"
	}

	return "a"
}
