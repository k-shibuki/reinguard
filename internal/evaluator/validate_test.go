package evaluator

import (
	"strings"
	"testing"
)

func TestValidateWhen_table(t *testing.T) {
	t.Parallel()
	// Given: when-clause trees and default registry
	// When: ValidateWhen runs per row
	// Then: nil or error substring per row
	reg := DefaultRegistry()
	tests := []struct {
		name    string
		when    any
		wantErr string
	}{
		{
			name: "unknown_nested_under_count",
			when: map[string]any{
				"op":   "count",
				"path": "items",
				"eq":   0,
				"when": map[string]any{"eval": "bogus"},
			},
			wantErr: "unknown evaluator",
		},
		{
			name: "eval_combined_with_op",
			when: map[string]any{
				"eval":  "constant",
				"op":    "eq",
				"path":  "a",
				"value": 1,
			},
			wantErr: "combine eval with op",
		},
		{
			name: "eval_combined_with_non_string_op",
			when: map[string]any{
				"eval": "constant",
				"op":   1,
			},
			wantErr: "combine eval with op",
		},
		{
			name: "eval_wrong_type",
			when: map[string]any{
				"eval": 1,
				"op":   "eq",
			},
			wantErr: "eval must be non-empty string",
		},
		{
			name: "eval_empty_string",
			when: map[string]any{
				"eval": "",
			},
			wantErr: "eval must be non-empty string",
		},
		{
			name: "valid_constant_only",
			when: map[string]any{
				"eval":   "constant",
				"params": map[string]any{"value": true},
			},
		},
		{
			name: "valid_nested_and",
			when: map[string]any{
				"and": []any{
					map[string]any{"op": "eq", "path": "x", "value": 1},
					map[string]any{"eval": "constant", "params": map[string]any{"value": true}},
				},
			},
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			err := ValidateWhen(tc.when, reg)
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err=%v want substring %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
