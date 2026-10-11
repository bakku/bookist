package optional_test

import (
	"encoding/json"
	"testing"

	"bakku.dev/bookist/internal/optional"
)

func TestValueDistinguishesOmittedNullAndValue(t *testing.T) {
	type request struct {
		Name optional.Value[string] `json:"name"`
	}

	var omitted request
	if err := json.Unmarshal([]byte(`{}`), &omitted); err != nil {
		t.Fatal(err)
	}
	if omitted.Name.Present {
		t.Fatal("expected omitted field not to be present")
	}

	var cleared request
	if err := json.Unmarshal([]byte(`{"name":null}`), &cleared); err != nil {
		t.Fatal(err)
	}
	if !cleared.Name.Present || cleared.Name.Value != nil {
		t.Fatalf("expected explicit null, got %#v", cleared.Name)
	}

	var set request
	if err := json.Unmarshal([]byte(`{"name":"Dune"}`), &set); err != nil {
		t.Fatal(err)
	}
	if !set.Name.Present || set.Name.Value == nil || *set.Name.Value != "Dune" {
		t.Fatalf("expected supplied value, got %#v", set.Name)
	}
}

// ── Marshal ───────────────────────────────────────────────────────────────────

func TestValueMarshalOmitsAbsentAndWritesNullAndValue(t *testing.T) {
	type request struct {
		Name  optional.Value[string]  `json:"name,omitzero"`
		Pages optional.Value[int]     `json:"pages,omitzero"`
		IDs   optional.Value[[]int64] `json:"ids,omitzero"`
	}

	tests := []struct {
		name     string
		request  request
		expected string
	}{
		{name: "absent", request: request{}, expected: `{}`},
		{name: "explicit null", request: request{Name: optional.Null[string](), IDs: optional.Null[[]int64]()}, expected: `{"name":null,"ids":null}`},
		{name: "value", request: request{Name: optional.Set("Dune"), Pages: optional.Set(412), IDs: optional.Set([]int64{4, 8})}, expected: `{"name":"Dune","pages":412,"ids":[4,8]}`},
		{name: "zero value is still written", request: request{Name: optional.Set(""), Pages: optional.Set(0)}, expected: `{"name":"","pages":0}`},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			encoded, err := json.Marshal(test.request)
			if err != nil {
				t.Fatal(err)
			}
			if string(encoded) != test.expected {
				t.Fatalf("expected %s, got %s", test.expected, encoded)
			}
		})
	}
}

func TestValueMarshalRoundTrips(t *testing.T) {
	type request struct {
		Name optional.Value[string] `json:"name,omitzero"`
	}

	for _, input := range []string{`{}`, `{"name":null}`, `{"name":"Dune"}`} {
		var decoded request
		if err := json.Unmarshal([]byte(input), &decoded); err != nil {
			t.Fatal(err)
		}
		encoded, err := json.Marshal(decoded)
		if err != nil {
			t.Fatal(err)
		}
		if string(encoded) != input {
			t.Fatalf("expected %s to round-trip, got %s", input, encoded)
		}
	}
}

func TestValueClearMarksPresentNull(t *testing.T) {
	value := optional.Set("Dune")
	value.Clear()

	if !value.Present || value.Value != nil {
		t.Fatalf("expected explicit null, got %#v", value)
	}
}
