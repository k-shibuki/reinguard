package evaluator

import (
	"slices"
	"testing"
)

func TestRegistry_Register_duplicate(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	if err := r.Register(constantEvaluator{}); err != nil {
		t.Fatal(err)
	}
	if err := r.Register(constantEvaluator{}); err == nil {
		t.Fatal("expected duplicate error")
	}
}

type emptyNameEval struct{}

func (emptyNameEval) Name() string { return "" }

func (emptyNameEval) Eval(map[string]any, map[string]any) (any, error) { return true, nil }

func TestRegistry_Register_emptyNameRejected(t *testing.T) {
	t.Parallel()
	r := NewRegistry()
	if err := r.Register(emptyNameEval{}); err == nil {
		t.Fatal("expected empty name error")
	}
}

func TestDefaultRegistry_ListRegistered_includesConstant(t *testing.T) {
	t.Parallel()
	names := DefaultRegistry().ListRegistered()
	if !slices.Contains(names, "constant") {
		t.Fatalf("expected constant in %v", names)
	}
	for i := 1; i < len(names); i++ {
		if names[i] < names[i-1] {
			t.Fatalf("not sorted: %v", names)
		}
	}
}

func TestRegistry_Get_nil(t *testing.T) {
	t.Parallel()
	e, ok := (*Registry)(nil).Get("x")
	if ok || e != nil {
		t.Fatalf("got %v %v", e, ok)
	}
}
