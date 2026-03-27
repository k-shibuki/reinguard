package match

import (
	"fmt"
	"strings"
	"testing"

	"github.com/k-shibuki/reinguard/internal/evaluator"
)

func requireMatchErr(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "match:") {
		t.Fatalf("error should contain match: prefix: %v", err)
	}
}

func TestEval_nilWhen(t *testing.T) {
	t.Parallel()
	// Given: nil when-clause
	// When: Eval is called
	// Then: error with match: prefix
	_, err := Eval(nil, map[string]any{})
	requireMatchErr(t, err)
	if !strings.Contains(err.Error(), "nil when") {
		t.Fatal(err)
	}
}

func TestEval_unsupportedType(t *testing.T) {
	t.Parallel()
	// Given: scalar when (invalid top-level shape)
	// When: Eval is called
	// Then: unsupported type error
	_, err := Eval("scalar", map[string]any{})
	requireMatchErr(t, err)
	if !strings.Contains(err.Error(), "unsupported when type") {
		t.Fatal(err)
	}
}

func TestEval_unknownMapShape(t *testing.T) {
	t.Parallel()
	// Given: map without op/and/or/not
	// When: Eval is called
	// Then: unknown when shape
	_, err := Eval(map[string]any{"foo": 1}, map[string]any{})
	requireMatchErr(t, err)
	if !strings.Contains(err.Error(), "unknown when shape") {
		t.Fatal(err)
	}
	if !strings.Contains(err.Error(), "eval") {
		t.Fatal("expected error to mention eval alternative:", err)
	}
}

func TestEval_unknownOp(t *testing.T) {
	t.Parallel()
	// Given: unknown operator name
	// When: Eval is called
	// Then: unknown op error
	_, err := Eval(map[string]any{"op": "bogus", "path": "a", "value": 1}, map[string]any{"a": 1})
	requireMatchErr(t, err)
	if !strings.Contains(err.Error(), "unknown op") {
		t.Fatal(err)
	}
}

func TestEval_andNotArray(t *testing.T) {
	t.Parallel()
	// Given: and with non-array value
	// When: Eval is called
	// Then: expected array error
	_, err := Eval(map[string]any{"and": 1}, map[string]any{})
	requireMatchErr(t, err)
	if !strings.Contains(err.Error(), "expected array") {
		t.Fatal(err)
	}
}

func TestEval_orNotArray(t *testing.T) {
	t.Parallel()
	// Given: or with non-array value
	// When: Eval is called
	_, err := Eval(map[string]any{"or": "x"}, map[string]any{})
	// Then: match error
	requireMatchErr(t, err)
}

func TestEval_missingPath(t *testing.T) {
	t.Parallel()
	// Given: eq without path
	// When: Eval is called
	_, err := Eval(map[string]any{"op": "eq", "value": 1}, map[string]any{})
	requireMatchErr(t, err)
	// Then: missing path error
	if !strings.Contains(err.Error(), "missing path") {
		t.Fatal(err)
	}
}

func TestEval_eqMissingValue(t *testing.T) {
	t.Parallel()
	// Given: eq without value
	// When: Eval is called
	_, err := Eval(map[string]any{"op": "eq", "path": "a"}, map[string]any{"a": 1})
	requireMatchErr(t, err)
	// Then: requires value error
	if !strings.Contains(err.Error(), "requires value") {
		t.Fatal(err)
	}
}

func TestEval_eq_numericSignal_nonNumericValue(t *testing.T) {
	t.Parallel()
	// Given: numeric signal compared to non-numeric string value (regression: no stack overflow)
	// When: Eval runs
	ok, err := Eval(map[string]any{"op": "eq", "path": "n", "value": "not-a-number"}, map[string]any{"n": 1})
	if err != nil {
		t.Fatal(err)
	}
	// Then: false, no error
	if ok {
		t.Fatal("expected false")
	}
}

func TestEval_inValueNotArray(t *testing.T) {
	t.Parallel()
	// Given: in-op with non-array value
	// When: Eval is called
	_, err := Eval(map[string]any{"op": "in", "path": "p", "value": "not-array"}, map[string]any{"p": "x"})
	requireMatchErr(t, err)
	// Then: validation error
	if !strings.Contains(err.Error(), "in value must be array") {
		t.Fatal(err)
	}
}

func TestEval_containsValueNotString(t *testing.T) {
	t.Parallel()
	// Given: contains-op with non-string value
	// When: Eval is called
	_, err := Eval(map[string]any{"op": "contains", "path": "a", "value": 1}, map[string]any{"a": "hi"})
	requireMatchErr(t, err)
	// Then: validation error
	if !strings.Contains(err.Error(), "contains value must be string") {
		t.Fatal(err)
	}
}

func TestEval_countRequiresEq(t *testing.T) {
	t.Parallel()
	// Given: count-op without eq
	// When: Eval is called
	_, err := Eval(map[string]any{"op": "count", "path": "xs"}, map[string]any{"xs": []any{1}})
	requireMatchErr(t, err)
	// Then: count requires eq error
	if !strings.Contains(err.Error(), "count requires eq") {
		t.Fatal(err)
	}
}

func TestEval_countEqMustBeInteger(t *testing.T) {
	t.Parallel()
	// Given: count with non-integer eq
	// When: Eval is called
	_, err := Eval(map[string]any{"op": "count", "path": "xs", "eq": 1.9}, map[string]any{"xs": []any{1, 2}})
	requireMatchErr(t, err)
	// Then: non-negative integer error
	if !strings.Contains(err.Error(), "non-negative integer") {
		t.Fatal(err)
	}
}

func TestEval_countOnTypedSlice(t *testing.T) {
	t.Parallel()
	// Given: count over []int-backed signal
	// When: Eval runs
	ok, err := Eval(map[string]any{"op": "count", "path": "xs", "eq": 2}, map[string]any{"xs": []int{1, 2}})
	if err != nil {
		t.Fatal(err)
	}
	// Then: true
	if !ok {
		t.Fatal("expected match on []int-backed signal")
	}
}

func TestEval_anyRequiresWhen(t *testing.T) {
	t.Parallel()
	// Given: any-op without nested when
	// When: Eval is called
	_, err := Eval(map[string]any{"op": "any", "path": "items"}, map[string]any{"items": []any{1}})
	requireMatchErr(t, err)
	// Then: any requires when error
	if !strings.Contains(err.Error(), "any requires when") {
		t.Fatal(err)
	}
}

func TestEval_allRequiresWhen(t *testing.T) {
	t.Parallel()
	// Given: all-op without nested when
	// When: Eval is called
	_, err := Eval(map[string]any{"op": "all", "path": "items"}, map[string]any{"items": []any{1}})
	requireMatchErr(t, err)
	// Then: all requires when error
	if !strings.Contains(err.Error(), "all requires when") {
		t.Fatal(err)
	}
}

func TestEval_cmpNumNonNumericValue(t *testing.T) {
	t.Parallel()
	// Given: numeric comparison with non-numeric value
	// When: Eval is called
	_, err := Eval(map[string]any{"op": "gt", "path": "a", "value": "x"}, map[string]any{"a": 1.0})
	requireMatchErr(t, err)
	// Then: value not numeric error
	if !strings.Contains(err.Error(), "value not numeric") {
		t.Fatal(err)
	}
}

func TestEvalOperatorMatrix(t *testing.T) {
	t.Parallel()
	// Given/When/Then: each row exercises Eval(tc.when, tc.sig) and expects tc.want
	tests := []struct {
		when any
		sig  map[string]any
		name string
		want bool
	}{
		{
			name: "eq_string_nested",
			when: map[string]any{"op": "eq", "path": "git.branch", "value": "main"},
			sig:  map[string]any{"git": map[string]any{"branch": "main"}},
			want: true,
		},
		{
			name: "ne",
			when: map[string]any{"op": "ne", "path": "a", "value": 2.0},
			sig:  map[string]any{"a": 1.0},
			want: true,
		},
		{
			name: "gt_nonnumeric_false",
			when: map[string]any{"op": "gt", "path": "a", "value": 1.0},
			sig:  map[string]any{"a": "x"},
			want: false,
		},
		{
			name: "in_member",
			when: map[string]any{"op": "in", "path": "phase", "value": []any{"review", "merge"}},
			sig:  map[string]any{"phase": "review"},
			want: true,
		},
		{
			name: "contains",
			when: map[string]any{"op": "contains", "path": "msg.t", "value": "world"},
			sig:  map[string]any{"msg": map[string]any{"t": "hello world"}},
			want: true,
		},
		{
			name: "exists",
			when: map[string]any{"op": "exists", "path": "a.b"},
			sig:  map[string]any{"a": map[string]any{"b": 1}},
			want: true,
		},
		{
			name: "not_exists",
			when: map[string]any{"op": "not_exists", "path": "b"},
			sig:  map[string]any{"a": 1},
			want: true,
		},
		{
			name: "count_raw",
			when: map[string]any{"op": "count", "path": "xs", "eq": 3},
			sig:  map[string]any{"xs": []any{1, 2, 3}},
			want: true,
		},
		{
			name: "bare_array_and",
			when: []any{
				map[string]any{"op": "eq", "path": "x", "value": true},
				map[string]any{"not": map[string]any{"op": "eq", "path": "y", "value": true}},
			},
			sig:  map[string]any{"x": true, "y": false},
			want: true,
		},
		{
			name: "or_short_circuit",
			when: map[string]any{"or": []any{
				map[string]any{"op": "eq", "path": "a", "value": true},
				map[string]any{"op": "eq", "path": "missing", "value": 1},
			}},
			sig:  map[string]any{"a": true},
			want: true,
		},
		{
			name: "all_empty_vacuous",
			when: map[string]any{
				"op": "all", "path": "items",
				"when": map[string]any{"op": "eq", "path": "$.x", "value": 1.0},
			},
			sig:  map[string]any{"items": []any{}},
			want: true,
		},
		{
			name: "any_empty_false",
			when: map[string]any{
				"op": "any", "path": "items",
				"when": map[string]any{"op": "eq", "path": "$.x", "value": 1.0},
			},
			sig:  map[string]any{"items": []any{}},
			want: false,
		},
		{
			name: "count_filtered",
			when: map[string]any{
				"op": "count", "path": "items",
				"when": map[string]any{"op": "eq", "path": "$.x", "value": 1.0},
				"eq":   1,
			},
			sig: map[string]any{"items": []any{
				map[string]any{"x": 1},
				map[string]any{"x": 2},
			}},
			want: true,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Given: tc.sig and tc.when (see name)
			// When: Eval runs
			// Then: want / wantErr
			ok, err := Eval(tc.when, tc.sig)
			if err != nil {
				t.Fatal(err)
			}
			if ok != tc.want {
				t.Fatalf("got %v want %v", ok, tc.want)
			}
		})
	}
}

func TestEval_namedEvaluator_constant(t *testing.T) {
	t.Parallel()
	// Given: constant true / false
	// When: Eval runs
	// Then: matches params.value
	ok, err := Eval(map[string]any{
		"eval":   "constant",
		"params": map[string]any{"value": true},
	}, map[string]any{})
	if err != nil || !ok {
		t.Fatalf("got %v %v", ok, err)
	}
	ok, err = Eval(map[string]any{
		"eval":   "constant",
		"params": map[string]any{"value": false},
	}, map[string]any{})
	if err != nil || ok {
		t.Fatalf("got %v %v", ok, err)
	}
}

func TestEval_namedEvaluator_unknown(t *testing.T) {
	t.Parallel()
	_, err := EvalWithRegistry(
		map[string]any{"eval": "nope", "params": map[string]any{}},
		map[string]any{},
		evaluator.NewRegistry(),
	)
	requireMatchErr(t, err)
	if !strings.Contains(err.Error(), "unknown evaluator") {
		t.Fatal(err)
	}
}

func TestEval_evalCombineOpError(t *testing.T) {
	t.Parallel()
	_, err := Eval(map[string]any{
		"eval":  "constant",
		"op":    "eq",
		"path":  "a",
		"value": 1,
	}, map[string]any{})
	requireMatchErr(t, err)
	if !strings.Contains(err.Error(), "combine eval with op") {
		t.Fatal(err)
	}
}

func TestEvalLteGteTable(t *testing.T) {
	t.Parallel()
	// Given: fixed signal n=5
	s := map[string]any{"n": 5.0}
	for _, tc := range []struct {
		op   string
		v    float64
		want bool
	}{
		{"lte", 5, true},
		{"lte", 4, false},
		{"gte", 5, true},
		{"gte", 6, false},
	} {
		t.Run(fmt.Sprintf("%s_%g", tc.op, tc.v), func(t *testing.T) {
			t.Parallel()
			// Given: signal n=5 and comparison op
			// When: Eval lte/gte
			// Then: want truth value
			when := map[string]any{"op": tc.op, "path": "n", "value": tc.v}
			ok, err := Eval(when, s)
			if err != nil {
				t.Fatal(err)
			}
			if ok != tc.want {
				t.Fatalf("%s %v: got %v", tc.op, tc.v, ok)
			}
		})
	}
}
