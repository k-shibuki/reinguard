package match

import "testing"

func TestEqScalar_table(t *testing.T) {
	t.Parallel()
	// Given/When/Then: each subtest compares two scalars with eqScalar
	tests := []struct {
		a    any
		b    any
		name string
		want bool
	}{
		{name: "bool_eq", a: true, b: true, want: true},
		{name: "bool_ne", a: true, b: false, want: false},
		{name: "bool_vs_string", a: true, b: "true", want: false},
		{name: "string_eq", a: "abc", b: "abc", want: true},
		{name: "string_ne", a: "abc", b: "xyz", want: false},
		{name: "int_eq_int", a: 5, b: 5, want: true},
		{name: "int_eq_float", a: 5, b: 5.0, want: true},
		{name: "int_ne_float", a: 5, b: 6.0, want: false},
		{name: "int64_eq_float", a: int64(10), b: 10.0, want: true},
		{name: "float_eq_int", a: 3.0, b: 3, want: true},
		{name: "float_ne_int", a: 3.5, b: 3, want: false},
		{name: "nil_nil", a: nil, b: nil, want: false},
		{name: "nil_string", a: nil, b: "x", want: false},
		{name: "int_vs_bool", a: 1, b: true, want: false},
		{name: "string_vs_int", a: "1", b: 1, want: false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Given: pair (tc.a, tc.b)
			// When: eqScalar runs
			// Then: result is tc.want
			if got := eqScalar(tc.a, tc.b); got != tc.want {
				t.Fatalf("eqScalar(%v, %v) = %v, want %v", tc.a, tc.b, got, tc.want)
			}
		})
	}
}

func TestEvalQuant_anyMatch(t *testing.T) {
	t.Parallel()
	// Given: any-quantifier over items where one element matches inner when
	when := map[string]any{
		"op": "any", "path": "items",
		"when": map[string]any{"op": "eq", "path": "$.status", "value": "ok"},
	}
	sig := map[string]any{
		"items": []any{
			map[string]any{"status": "fail"},
			map[string]any{"status": "ok"},
		},
	}
	// When: Eval runs
	ok, err := Eval(when, sig)
	if err != nil {
		t.Fatal(err)
	}
	// Then: true
	if !ok {
		t.Fatal("expected any-match to be true")
	}
}

func TestEvalQuant_anyNoMatch(t *testing.T) {
	t.Parallel()
	// Given: any-quantifier where no element matches
	when := map[string]any{
		"op": "any", "path": "items",
		"when": map[string]any{"op": "eq", "path": "$.status", "value": "ok"},
	}
	sig := map[string]any{
		"items": []any{
			map[string]any{"status": "fail"},
			map[string]any{"status": "error"},
		},
	}
	// When: Eval runs
	ok, err := Eval(when, sig)
	if err != nil {
		t.Fatal(err)
	}
	// Then: false
	if ok {
		t.Fatal("expected any-match to be false")
	}
}

func TestEvalQuant_allMatch(t *testing.T) {
	t.Parallel()
	// Given: all-quantifier where every element satisfies inner when
	when := map[string]any{
		"op": "all", "path": "items",
		"when": map[string]any{"op": "gt", "path": "$.v", "value": 0.0},
	}
	sig := map[string]any{
		"items": []any{
			map[string]any{"v": 1.0},
			map[string]any{"v": 2.0},
		},
	}
	// When: Eval runs
	ok, err := Eval(when, sig)
	if err != nil {
		t.Fatal(err)
	}
	// Then: true
	if !ok {
		t.Fatal("expected all-match to be true")
	}
}

func TestEvalQuant_allPartialFail(t *testing.T) {
	t.Parallel()
	// Given: all-quantifier where one element fails inner when
	when := map[string]any{
		"op": "all", "path": "items",
		"when": map[string]any{"op": "gt", "path": "$.v", "value": 0.0},
	}
	sig := map[string]any{
		"items": []any{
			map[string]any{"v": 1.0},
			map[string]any{"v": -1.0},
		},
	}
	// When: Eval runs
	ok, err := Eval(when, sig)
	if err != nil {
		t.Fatal(err)
	}
	// Then: false
	if ok {
		t.Fatal("expected all-match to be false with partial failure")
	}
}

func TestEvalQuant_nonArrayPath(t *testing.T) {
	t.Parallel()
	// Given: any-op but path resolves to a non-array scalar
	when := map[string]any{
		"op": "any", "path": "x",
		"when": map[string]any{"op": "eq", "path": "$.v", "value": 1.0},
	}
	// When: Eval runs
	ok, err := Eval(when, map[string]any{"x": "scalar"})
	if err != nil {
		t.Fatal(err)
	}
	// Then: false (no match over non-iterable)
	if ok {
		t.Fatal("expected false for non-array path")
	}
}

func TestEvalIn_notMember(t *testing.T) {
	t.Parallel()
	// Given: in-op with signal value not in list
	when := map[string]any{"op": "in", "path": "p", "value": []any{"a", "b", "c"}}
	// When: Eval runs
	ok, err := Eval(when, map[string]any{"p": "z"})
	if err != nil {
		t.Fatal(err)
	}
	// Then: false
	if ok {
		t.Fatal("expected not-in-list")
	}
}

func TestEvalIn_missingPath(t *testing.T) {
	t.Parallel()
	// Given: in-op on missing path
	when := map[string]any{"op": "in", "path": "missing", "value": []any{"a"}}
	// When: Eval runs
	ok, err := Eval(when, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	// Then: false
	if ok {
		t.Fatal("expected false for missing path")
	}
}

func TestEvalContains_notFound(t *testing.T) {
	t.Parallel()
	// Given: contains-op where substring is absent
	when := map[string]any{"op": "contains", "path": "s", "value": "xyz"}
	// When: Eval runs
	ok, err := Eval(when, map[string]any{"s": "hello world"})
	if err != nil {
		t.Fatal(err)
	}
	// Then: false
	if ok {
		t.Fatal("expected false for not-contains")
	}
}

func TestEvalContains_missingPath(t *testing.T) {
	t.Parallel()
	// Given: contains-op on missing path
	when := map[string]any{"op": "contains", "path": "missing", "value": "x"}
	// When: Eval runs
	ok, err := Eval(when, map[string]any{})
	if err != nil {
		t.Fatal(err)
	}
	// Then: false
	if ok {
		t.Fatal("expected false for missing path")
	}
}

func TestEvalContains_nonStringSignal(t *testing.T) {
	t.Parallel()
	// Given: contains-op but signal at path is not a string
	when := map[string]any{"op": "contains", "path": "n", "value": "1"}
	// When: Eval runs
	ok, err := Eval(when, map[string]any{"n": 123})
	if err != nil {
		t.Fatal(err)
	}
	// Then: false
	if ok {
		t.Fatal("expected false for non-string signal")
	}
}

func TestEvalExists_false(t *testing.T) {
	t.Parallel()
	// Given: exists-op on missing nested path
	when := map[string]any{"op": "exists", "path": "missing.deep"}
	// When: Eval runs
	ok, err := Eval(when, map[string]any{"other": 1})
	if err != nil {
		t.Fatal(err)
	}
	// Then: false
	if ok {
		t.Fatal("expected false for missing path")
	}
}

func TestEvalNotExists_found(t *testing.T) {
	t.Parallel()
	// Given: not_exists-op but path is present
	when := map[string]any{"op": "not_exists", "path": "x"}
	// When: Eval runs
	ok, err := Eval(when, map[string]any{"x": 1})
	if err != nil {
		t.Fatal(err)
	}
	// Then: false
	if ok {
		t.Fatal("expected false when path exists")
	}
}

func TestEvalNot(t *testing.T) {
	t.Parallel()
	// Given: not wrapping eq that would be false on signals
	when := map[string]any{"not": map[string]any{"op": "eq", "path": "x", "value": 1}}
	// When: Eval runs
	ok, err := Eval(when, map[string]any{"x": 2})
	if err != nil {
		t.Fatal(err)
	}
	// Then: true (negated false)
	if !ok {
		t.Fatal("expected not(eq) to be true")
	}
}

func TestEvalGt_numericSignalTrue(t *testing.T) {
	t.Parallel()
	// Given: gt when signal exceeds threshold
	when := map[string]any{"op": "gt", "path": "n", "value": 5.0}
	// When: Eval runs
	ok, err := Eval(when, map[string]any{"n": 10.0})
	if err != nil {
		t.Fatal(err)
	}
	// Then: true
	if !ok {
		t.Fatal("expected true")
	}
}

func TestEvalLt_numericSignalTrue(t *testing.T) {
	t.Parallel()
	// Given: lt when signal is below threshold
	when := map[string]any{"op": "lt", "path": "n", "value": 5.0}
	// When: Eval runs
	ok, err := Eval(when, map[string]any{"n": 3.0})
	if err != nil {
		t.Fatal(err)
	}
	// Then: true
	if !ok {
		t.Fatal("expected true")
	}
}

func TestToFloat_string(t *testing.T) {
	t.Parallel()
	// Given: numeric string
	// When: toFloat parses it
	f, err := toFloat("3.14")
	if err != nil {
		t.Fatal(err)
	}
	// Then: float matches
	if f != 3.14 {
		t.Fatalf("got %v", f)
	}
}

func TestToFloat_unsupported(t *testing.T) {
	t.Parallel()
	// Given: non-numeric value
	// When: toFloat is called
	_, err := toFloat(true)
	// Then: error
	if err == nil {
		t.Fatal("expected error for bool")
	}
}
