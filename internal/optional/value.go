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

// Set returns a Value carrying value.
func Set[T any](value T) Value[T] {
	return Value[T]{Present: true, Value: &value}
}

// Null returns a Value that is present but explicitly null.
func Null[T any]() Value[T] {
	return Value[T]{Present: true}
}

// Clear marks the value present and explicitly null.
func (v *Value[T]) Clear() {
	v.Present = true
	v.Value = nil
}

// IsZero reports whether the value is absent, so that omitzero drops it.
func (v Value[T]) IsZero() bool {
	return !v.Present
}

// MarshalJSON writes an explicit null for a cleared value. Absent values are
// expected to be dropped by the omitzero JSON tag option; marshaled directly
// they are written as null.
func (v Value[T]) MarshalJSON() ([]byte, error) {
	if v.Value == nil {
		return []byte("null"), nil
	}
	return json.Marshal(*v.Value)
}
