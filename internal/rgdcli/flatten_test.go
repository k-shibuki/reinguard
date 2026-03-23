package rgdcli

import "testing"

func TestFlattenSignals_nested(t *testing.T) {
	t.Parallel()
	// Given: namespaced signals with nesting
	s := map[string]any{
		"github": map[string]any{
			"ci": map[string]any{"ci_status": "success"},
		},
	}
	// When: flattenSignals runs
	out := flattenSignals(s)
	// Then: dotted keys exist for match rules
	if out["github.ci.ci_status"] != "success" {
		t.Fatalf("%v", out)
	}
}

func TestFlattenSignals_topLevelScalar(t *testing.T) {
	t.Parallel()
	out := flattenSignals(map[string]any{"x": 42})
	v, ok := out["x"].(int)
	if !ok || v != 42 {
		t.Fatalf("%v", out["x"])
	}
}
