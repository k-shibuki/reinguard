package observe

import (
	"encoding/json"
	"testing"
)

func TestParseObservationJSON_diagnosticsNonStringFieldsOmitted(t *testing.T) {
	t.Parallel()
	// Given: diagnostics with non-string severity (invalid type coerced to empty)
	data, err := json.Marshal(map[string]any{
		"signals": map[string]any{"x": 1},
		"diagnostics": []any{
			map[string]any{"severity": 123, "message": "hello", "provider": "p", "code": "c"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	// When: ParseObservationJSON runs
	_, diags, _, err := ParseObservationJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	// Then: string fields are empty when JSON types are not strings
	if len(diags) != 1 {
		t.Fatalf("got %d diags", len(diags))
	}
	if diags[0].Severity != "" || diags[0].Message != "hello" {
		t.Fatalf("got severity=%q message=%q", diags[0].Severity, diags[0].Message)
	}
}

func TestParseObservationJSON_stringDiagnostics(t *testing.T) {
	t.Parallel()
	data, err := json.Marshal(map[string]any{
		"signals": map[string]any{"a": true},
		"diagnostics": []any{
			map[string]any{"severity": "warn", "message": "m", "provider": "git", "code": "x"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, diags, _, err := ParseObservationJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(diags) != 1 || diags[0].Severity != "warn" || diags[0].Message != "m" || diags[0].Provider != "git" || diags[0].Code != "x" {
		t.Fatalf("unexpected diags: %+v", diags)
	}
}
