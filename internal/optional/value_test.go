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
