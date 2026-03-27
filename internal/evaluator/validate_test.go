package evaluator

import "testing"

func TestValidateWhen_unknownNestedInCount(t *testing.T) {
	t.Parallel()
	when := map[string]any{
		"op":   "count",
		"path": "items",
		"eq":   0,
		"when": map[string]any{"eval": "bogus"},
	}
	err := ValidateWhen(when, DefaultRegistry())
	if err == nil || err.Error() == "" {
		t.Fatal("expected unknown evaluator")
	}
}

func TestValidateWhen_evalWithOpRejected(t *testing.T) {
	t.Parallel()
	when := map[string]any{
		"eval":  "constant",
		"op":    "eq",
		"path":  "a",
		"value": 1,
	}
	err := ValidateWhen(when, DefaultRegistry())
	if err == nil || err.Error() == "" {
		t.Fatal("expected combine error")
	}
}
