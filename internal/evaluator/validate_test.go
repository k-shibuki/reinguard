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
				"path": "state.items",
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
				"path":  "git.branch",
				"value": "main",
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
					map[string]any{"op": "eq", "path": "git.branch", "value": "main"},
					map[string]any{"eval": "constant", "params": map[string]any{"value": true}},
				},
			},
		},
		{
			name:    "unknown_op",
			when:    map[string]any{"op": "bogus", "path": "git.branch", "value": 1},
			wantErr: "unknown op",
		},
		{
			name:    "path_bad_prefix",
			when:    map[string]any{"op": "eq", "path": "foo.bar", "value": 1},
			wantErr: "must start with git.",
		},
		{
			name:    "eq_missing_value",
			when:    map[string]any{"op": "eq", "path": "git.branch"},
			wantErr: "requires value",
		},
		{
			name:    "constant_missing_params",
			when:    map[string]any{"eval": "constant"},
			wantErr: "requires params object",
		},
		{
			name: "path_dollar_quantifier_ok",
			when: map[string]any{
				"op": "any", "path": "state.items", "when": map[string]any{"op": "eq", "path": "$.x", "value": 1.0},
			},
		},
		{
			name:    "empty_object",
			when:    map[string]any{},
			wantErr: "unknown shape",
		},
		{
			name:    "scalar_string",
			when:    "nope",
			wantErr: "object or array",
		},
		{
			name:    "path_leading_whitespace",
			when:    map[string]any{"op": "eq", "path": " github.repository.owner", "value": "x"},
			wantErr: "leading or trailing whitespace",
		},
		{
			name:    "not_child_nil",
			when:    map[string]any{"not": nil},
			wantErr: "nil",
		},
		{
			name:    "and_child_nil",
			when:    map[string]any{"and": []any{nil}},
			wantErr: "nil",
		},
		{
			name: "any_when_nil",
			when: map[string]any{
				"op":   "any",
				"path": "state.items",
				"when": nil,
			},
			wantErr: "nil",
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
