package optional

import (
	"bytes"
	"encoding/json"
)

// Value distinguishes an omitted JSON field from a value and an explicit null.
type Value[T any] struct {
	Present bool
	Value   *T
}

func (v *Value[T]) UnmarshalJSON(data []byte) error {
	v.Present = true
	if bytes.Equal(bytes.TrimSpace(data), []byte("null")) {
		v.Value = nil
		return nil
	}

	var value T
	if err := json.Unmarshal(data, &value); err != nil {
		return err
	}
	v.Value = &value
	return nil
}
