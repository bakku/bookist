package cli

import (
	"net/url"
	"strings"
)

func joinURL(base string, path string) (string, error) {
	parsed, err := url.Parse(base)
	if err != nil {
		return "", err
	}

	parsed.Path = strings.TrimRight(parsed.Path, "/") + path

	return parsed.String(), nil
}

func joinURLWithQuery(base, path, query string) (string, error) {
	endpoint, err := joinURL(base, path)
	if err != nil {
		return "", err
	}

	parsed, err := url.Parse(endpoint)
	if err != nil {
		return "", err
	}
	if strings.TrimSpace(query) != "" {
		values := parsed.Query()
		values.Set("q", query)
		parsed.RawQuery = values.Encode()
	}

	return parsed.String(), nil
}
