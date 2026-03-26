package signals

import "testing"

func TestFlatten(t *testing.T) {
	t.Parallel()
	tests := []struct {
		wantVal any
		input   map[string]any
		name    string
		wantKey string
		wantMap bool
		nilVal  bool
	}{
		{
			name:    "nested dotted key",
			input:   map[string]any{"github": map[string]any{"ci": map[string]any{"ci_status": "success"}}},
			wantKey: "github.ci.ci_status",
			wantVal: "success",
		},
		{
			name:    "intermediate key preserved as map",
			input:   map[string]any{"github": map[string]any{"ci": map[string]any{"ci_status": "success"}}},
			wantKey: "github.ci",
			wantMap: true,
		},
		{
			name:    "top-level scalar",
			input:   map[string]any{"x": 42},
			wantKey: "x",
			wantVal: 42,
		},
		{
			name:  "empty map produces empty output",
			input: map[string]any{},
		},
		{
			name:    "nil value preserved",
			input:   map[string]any{"foo": nil},
			wantKey: "foo",
			nilVal:  true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Given: input signals from test case
			// When: Flatten runs
			out := Flatten(tc.input)
			// Then: output matches expected key/value
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
			if tc.wantMap {
				if _, ok := got.(map[string]any); !ok {
					t.Fatalf("key %q: got %T, want map[string]any", tc.wantKey, got)
				}
				return
			}
			if tc.nilVal {
				if got != nil {
					t.Fatalf("key %q: got %v, want nil", tc.wantKey, got)
				}
				return
			}
			if got != tc.wantVal {
				t.Fatalf("key %q: got %v, want %v", tc.wantKey, got, tc.wantVal)
			}
		})
	}
}
