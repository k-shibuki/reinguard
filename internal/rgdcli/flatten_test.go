package rgdcli

import "testing"

func TestFlattenSignals(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   map[string]any
		wantKey string
		wantVal any
	}{
		{
			name:    "nested dotted key",
			input:   map[string]any{"github": map[string]any{"ci": map[string]any{"ci_status": "success"}}},
			wantKey: "github.ci.ci_status",
			wantVal: "success",
		},
		{
			name:    "intermediate key preserved",
			input:   map[string]any{"github": map[string]any{"ci": map[string]any{"ci_status": "success"}}},
			wantKey: "github.ci",
		},
		{
			name:    "top-level scalar",
			input:   map[string]any{"x": 42},
			wantKey: "x",
			wantVal: 42,
		},
		{
			name:  "empty map",
			input: map[string]any{},
		},
		{
			name:    "nil value preserved",
			input:   map[string]any{"foo": nil},
			wantKey: "foo",
			wantVal: nil,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			out := flattenSignals(tc.input)
			if tc.wantKey == "" {
				if len(out) != 0 {
					t.Fatalf("expected empty output, got %v", out)
				}
				return
			}
			got, ok := out[tc.wantKey]
			if !ok {
				t.Fatalf("key %q not found in %v", tc.wantKey, out)
			}
			if tc.wantVal != nil && got != tc.wantVal {
				t.Fatalf("key %q: got %v, want %v", tc.wantKey, got, tc.wantVal)
			}
		})
	}
}
