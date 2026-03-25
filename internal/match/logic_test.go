package match

import (
	"strings"
	"testing"
)

func TestEvalAnd_failingClause(t *testing.T) {
	t.Parallel()
	when := map[string]any{"and": []any{
		map[string]any{"op": "eq", "path": "a", "value": 1},
		map[string]any{"op": "eq", "path": "b", "value": 2},
	}}
	ok, err := Eval(when, map[string]any{"a": 1, "b": 99})
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected false when one clause fails")
	}
}

func TestEvalAnd_errorPropagation(t *testing.T) {
	t.Parallel()
	when := map[string]any{"and": []any{
		map[string]any{"op": "bogus", "path": "a"},
	}}
	_, err := Eval(when, map[string]any{"a": 1})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestEvalOr_allFalse(t *testing.T) {
	t.Parallel()
	when := map[string]any{"or": []any{
		map[string]any{"op": "eq", "path": "a", "value": 99},
		map[string]any{"op": "eq", "path": "a", "value": 100},
	}}
	ok, err := Eval(when, map[string]any{"a": 1})
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected false when all or clauses fail")
	}
}

func TestEvalOr_errorPropagation(t *testing.T) {
	t.Parallel()
	when := map[string]any{"or": []any{
		map[string]any{"op": "eq", "path": "a", "value": 99},
		map[string]any{"op": "bogus"},
	}}
	_, err := Eval(when, map[string]any{"a": 1})
	if err == nil || !strings.Contains(err.Error(), "match:") {
		t.Fatalf("expected match error, got %v", err)
	}
}

func TestEvalNe_equal(t *testing.T) {
	t.Parallel()
	when := map[string]any{"op": "ne", "path": "a", "value": 5.0}
	ok, err := Eval(when, map[string]any{"a": 5.0})
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected false for ne with equal values")
	}
}

func TestEvalEq_missingPath(t *testing.T) {
	t.Parallel()
	when := map[string]any{"op": "eq", "path": "missing", "value": 1}
	ok, err := Eval(when, map[string]any{"other": 1})
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected false for missing signal path")
	}
}

func TestEvalNe_missingPath(t *testing.T) {
	t.Parallel()
	when := map[string]any{"op": "ne", "path": "missing", "value": 1}
	ok, err := Eval(when, map[string]any{"other": 1})
	if err != nil {
		t.Fatal(err)
	}
	if ok {
		t.Fatal("expected false for ne with missing signal path")
	}
}
