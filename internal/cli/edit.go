package cli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

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
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("server returned %s", resp.Status)
	}
	if err := json.NewDecoder(resp.Body).Decode(output); err != nil {
		return fmt.Errorf("decode response: %v", err)
	}
	return nil
}
