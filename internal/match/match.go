// Package match evaluates ADR-0002 match expressions against signal maps.
package match

import (
	"fmt"
	"math"
	"reflect"
	"strconv"
	"strings"
)

// Eval evaluates a when-clause (YAML/JSON decoded as map, slice, or scalar)
// against signals. Dot paths address nested maps (e.g. "git.branch").
func Eval(when any, signals map[string]any) (bool, error) {
	if when == nil {
		return false, fmt.Errorf("match: nil when")
	}
	return evalValue(when, signals)
}

func evalValue(v any, signals map[string]any) (bool, error) {
	switch t := v.(type) {
	case map[string]any:
		return evalMap(t, signals)
	case []any:
		// Treat bare array as AND of elements (convenience).
		return evalAnd(t, signals)
	default:
		return false, fmt.Errorf("match: unsupported when type %T", v)
	}
}

func evalMap(m map[string]any, signals map[string]any) (bool, error) {
	if op, ok := m["op"].(string); ok {
		return evalOp(op, m, signals)
	}
	if v, ok := m["and"]; ok {
		arr, err := asSlice(v)
		if err != nil {
			return false, err
		}
		return evalAnd(arr, signals)
	}
	if v, ok := m["or"]; ok {
		arr, err := asSlice(v)
		if err != nil {
			return false, err
		}
		return evalOr(arr, signals)
	}
	if v, ok := m["not"]; ok {
		inner, err := evalValue(v, signals)
		if err != nil {
			return false, err
		}
		return !inner, nil
	}
	return false, fmt.Errorf("match: unknown when shape (missing op/and/or/not)")
}

func evalOp(op string, m map[string]any, signals map[string]any) (bool, error) {
	switch op {
	case "eq":
		return cmpScalar(m, signals, func(c int) bool { return c == 0 })
	case "ne":
		return cmpScalar(m, signals, func(c int) bool { return c != 0 })
	case "gt":
		return cmpNum(m, signals, func(a, b float64) bool { return a > b })
	case "lt":
		return cmpNum(m, signals, func(a, b float64) bool { return a < b })
	case "gte":
		return cmpNum(m, signals, func(a, b float64) bool { return a >= b })
	case "lte":
		return cmpNum(m, signals, func(a, b float64) bool { return a <= b })
	case "in":
		return evalIn(m, signals)
	case "contains":
		return evalContains(m, signals)
	case "exists":
		return evalExists(m, signals, true)
	case "not_exists":
		return evalExists(m, signals, false)
	case "count":
		return evalCount(m, signals)
	case "any":
		return evalQuant("any", m, signals)
	case "all":
		return evalQuant("all", m, signals)
	default:
		return false, fmt.Errorf("match: unknown op %q", op)
	}
}

func evalAnd(nodes []any, signals map[string]any) (bool, error) {
	for _, n := range nodes {
		ok, err := evalValue(n, signals)
		if err != nil {
			return false, err
		}
		if !ok {
			return false, nil
		}
	}
	return true, nil
}

func evalOr(nodes []any, signals map[string]any) (bool, error) {
	for _, n := range nodes {
		ok, err := evalValue(n, signals)
		if err != nil {
			return false, err
		}
		if ok {
			return true, nil
		}
	}
	return false, nil
}

func cmpScalar(m map[string]any, signals map[string]any, pred func(int) bool) (bool, error) {
	path, err := pathFrom(m)
	if err != nil {
		return false, err
	}
	want, ok := m["value"]
	if !ok {
		return false, fmt.Errorf("match: %s requires value", m["op"])
	}
	got, ok := getPath(signals, path)
	if !ok {
		return false, nil
	}
	return pred(compareValues(got, want)), nil
}

func cmpNum(m map[string]any, signals map[string]any, pred func(a, b float64) bool) (bool, error) {
	path, err := pathFrom(m)
	if err != nil {
		return false, err
	}
	want, ok := m["value"]
	if !ok {
		return false, fmt.Errorf("match: numeric op requires value")
	}
	got, ok := getPath(signals, path)
	if !ok {
		return false, nil
	}
	ga, err := toFloat(got)
	if err != nil {
		return false, nil
	}
	gb, err := toFloat(want)
	if err != nil {
		return false, fmt.Errorf("match: value not numeric: %w", err)
	}
	return pred(ga, gb), nil
}

func evalIn(m map[string]any, signals map[string]any) (bool, error) {
	path, err := pathFrom(m)
	if err != nil {
		return false, err
	}
	hay, ok := m["value"]
	if !ok {
		return false, fmt.Errorf("match: in requires value (array)")
	}
	arr, err := asSlice(hay)
	if err != nil {
		return false, fmt.Errorf("match: in value must be array: %w", err)
	}
	needle, ok := getPath(signals, path)
	if !ok {
		return false, nil
	}
	for _, elt := range arr {
		if compareValues(needle, elt) == 0 {
			return true, nil
		}
	}
	return false, nil
}

func evalContains(m map[string]any, signals map[string]any) (bool, error) {
	path, err := pathFrom(m)
	if err != nil {
		return false, err
	}
	sub, ok := m["value"]
	if !ok {
		return false, fmt.Errorf("match: contains requires value")
	}
	got, ok := getPath(signals, path)
	if !ok {
		return false, nil
	}
	gs, ok := got.(string)
	if !ok {
		return false, nil
	}
	ss, ok := sub.(string)
	if !ok {
		return false, fmt.Errorf("match: contains value must be string")
	}
	return strings.Contains(gs, ss), nil
}

func evalExists(m map[string]any, signals map[string]any, wantExists bool) (bool, error) {
	path, err := pathFrom(m)
	if err != nil {
		return false, err
	}
	_, ok := getPath(signals, path)
	if wantExists {
		return ok, nil
	}
	return !ok, nil
}

func evalCount(m map[string]any, signals map[string]any) (bool, error) {
	path, err := pathFrom(m)
	if err != nil {
		return false, err
	}
	got, ok := getPath(signals, path)
	if !ok {
		return false, nil
	}
	arr, ok := asSignalSlice(got)
	if !ok {
		return false, nil
	}
	n := len(arr)
	sub, ok := m["when"]
	if !ok {
		// No sub-filter: compare raw count.
		exp, expErr := expectCount(m)
		if expErr != nil {
			return false, expErr
		}
		return n == exp, nil
	}
	cnt := 0
	for _, elt := range arr {
		sig := map[string]any{"$": elt}
		ok, subErr := evalValue(sub, sig)
		if subErr != nil {
			return false, subErr
		}
		if ok {
			cnt++
		}
	}
	exp, err := expectCount(m)
	if err != nil {
		return false, err
	}
	return cnt == exp, nil
}

func expectCount(m map[string]any) (int, error) {
	if v, ok := m["eq"]; ok {
		f, err := toFloat(v)
		if err != nil {
			return 0, err
		}
		if f < 0 || math.Trunc(f) != f {
			return 0, fmt.Errorf("match: count eq must be a non-negative integer")
		}
		return int(f), nil
	}
	return 0, fmt.Errorf("match: count requires eq")
}

func evalQuant(kind string, m map[string]any, signals map[string]any) (bool, error) {
	path, err := pathFrom(m)
	if err != nil {
		return false, err
	}
	sub, ok := m["when"]
	if !ok {
		return false, fmt.Errorf("match: %s requires when", kind)
	}
	got, ok := getPath(signals, path)
	if !ok {
		return false, nil
	}
	arr, ok := asSignalSlice(got)
	if !ok {
		return false, nil
	}
	if len(arr) == 0 {
		// Vacuous truth for all; any is false.
		if kind == "all" {
			return true, nil
		}
		return false, nil
	}
	for _, elt := range arr {
		sig := map[string]any{"$": elt}
		ok, err := evalValue(sub, sig)
		if err != nil {
			return false, err
		}
		if kind == "any" && ok {
			return true, nil
		}
		if kind == "all" && !ok {
			return false, nil
		}
	}
	return kind == "all", nil
}

func pathFrom(m map[string]any) (string, error) {
	p, ok := m["path"].(string)
	if !ok || p == "" {
		return "", fmt.Errorf("match: missing path")
	}
	return p, nil
}

func asSlice(v any) ([]any, error) {
	switch t := v.(type) {
	case []any:
		return t, nil
	default:
		return nil, fmt.Errorf("match: expected array, got %T", v)
	}
}

// asSignalSlice accepts []any and other slice/array kinds produced by observation or tests.
func asSignalSlice(v any) ([]any, bool) {
	if v == nil {
		return nil, false
	}
	rv := reflect.ValueOf(v)
	switch rv.Kind() {
	case reflect.Slice, reflect.Array:
		n := rv.Len()
		out := make([]any, n)
		for i := 0; i < n; i++ {
			out[i] = rv.Index(i).Interface()
		}
		return out, true
	default:
		return nil, false
	}
}

func getPath(root map[string]any, path string) (any, bool) {
	parts := strings.Split(path, ".")
	var cur any = root
	for _, p := range parts {
		// Ignore empty segments so "a..b" addresses the same path as "a.b"
		// (resilient to accidental double dots in config).
		if p == "" {
			continue
		}
		mm, ok := cur.(map[string]any)
		if !ok {
			return nil, false
		}
		v, ok := mm[p]
		if !ok {
			return nil, false
		}
		cur = v
	}
	return cur, true
}

func compareValues(a, b any) int {
	// Numeric comparison if both convert to float.
	if fa, err1 := toFloat(a); err1 == nil {
		if fb, err2 := toFloat(b); err2 == nil {
			if fa < fb {
				return -1
			}
			if fa > fb {
				return 1
			}
			return 0
		}
	}
	as, aok := a.(string)
	bs, bok := b.(string)
	if aok && bok {
		return strings.Compare(as, bs)
	}
	if eqScalar(a, b) {
		return 0
	}
	return 1
}

func eqScalar(a, b any) bool {
	switch av := a.(type) {
	case bool:
		bv, ok := b.(bool)
		return ok && av == bv
	case string:
		bv, ok := b.(string)
		return ok && av == bv
	case int:
		return numericEqual(float64(av), b)
	case int64:
		return numericEqual(float64(av), b)
	case float64:
		return numericEqual(av, b)
	default:
		return false
	}
}

// numericEqual reports whether a equals b when b is numeric; non-numeric b is false.
// Used by eqScalar so we never recurse through compareValues (which would call eqScalar again).
func numericEqual(a float64, b any) bool {
	fb, err := toFloat(b)
	if err != nil {
		return false
	}
	return a == fb
}

func toFloat(v any) (float64, error) {
	switch x := v.(type) {
	case float64:
		return x, nil
	case int:
		return float64(x), nil
	case int64:
		return float64(x), nil
	case string:
		return strconv.ParseFloat(x, 64)
	default:
		return 0, fmt.Errorf("not numeric")
	}
}
