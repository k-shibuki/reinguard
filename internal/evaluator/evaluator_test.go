package evaluator

import (
	"slices"
	"strings"
	"testing"
)

func TestRegistry_Register_table(t *testing.T) {
	t.Parallel()
	//nolint:govet // table-driven case struct; fieldalignment is not worth obfuscating field order
	tests := []struct {
		name    string
		nilRecv bool
		prep    func(*Registry) error
		second  Evaluator
		wantErr string
	}{
		{
			name:    "nil_registry",
			nilRecv: true,
			second:  constantEvaluator{},
			wantErr: "nil registry",
		},
		{
			name:    "duplicate_name",
			prep:    func(r *Registry) error { return r.Register(constantEvaluator{}) },
			second:  constantEvaluator{},
			wantErr: "already registered",
		},
		{
			name:    "empty_name",
			second:  emptyNameEval{},
			wantErr: "empty name",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if tc.nilRecv {
				var r *Registry
				err := r.Register(tc.second)
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("err=%v want substring %q", err, tc.wantErr)
				}
				return
			}
			r := NewRegistry()
			if tc.prep != nil {
				if err := tc.prep(r); err != nil {
					t.Fatal(err)
				}
			}
			err := r.Register(tc.second)
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

type emptyNameEval struct{}

func (emptyNameEval) Name() string { return "" }

func (emptyNameEval) Eval(map[string]any, map[string]any) (any, error) { return true, nil }

func TestDefaultRegistry_ListRegistered_sortedIncludesConstant(t *testing.T) {
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

func TestRegistry_Get_nilReceiver(t *testing.T) {
	t.Parallel()
	e, ok := (*Registry)(nil).Get("x")
	if ok || e != nil {
		t.Fatalf("got %v %v", e, ok)
	}
}
