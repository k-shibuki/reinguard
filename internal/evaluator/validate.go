package evaluator

import (
	"fmt"
	"strings"
)

// knownWhenOps matches internal/match.evalOp switch (ADR-0002).
var knownWhenOps = map[string]struct{}{
	"eq": {}, "ne": {}, "gt": {}, "lt": {}, "gte": {}, "lte": {},
	"in": {}, "contains": {}, "exists": {}, "not_exists": {},
	"count": {}, "any": {}, "all": {},
}

// ValidateWhen checks when-clauses for config load: registered evaluators, eval/op shape,
// known operators, required operands per op, and allowed signal path prefixes (git., github., state., $).
func ValidateWhen(when any, reg *Registry) error {
	if reg == nil {
		reg = DefaultRegistry()
	}
	if when == nil {
		return nil
	}
	return walkWhen(when, reg)
}

func walkWhen(when any, reg *Registry) error {
	if when == nil {
		return fmt.Errorf("when clause is nil")
	}
	switch t := when.(type) {
	case map[string]any:
		return walkWhenMap(t, reg)
	case []any:
		for _, elt := range t {
			if err := walkWhen(elt, reg); err != nil {
				return err
			}
		}
		return nil
	default:
		return fmt.Errorf("when clause must be object or array, got %T", when)
	}
}

func walkWhenMap(m map[string]any, reg *Registry) error {
	if raw, hasEval := m["eval"]; hasEval {
		name, ok := raw.(string)
		if !ok || name == "" {
			return fmt.Errorf("when clause eval must be non-empty string")
		}
		if err := visitEvalName(name, reg); err != nil {
			return err
		}
		if err := rejectEvalCombiners(m); err != nil {
			return err
		}
		if name == "constant" {
			if err := validateConstantParams(m); err != nil {
				return err
			}
		}
		return nil
	}

	if raw, hasOp := m["op"]; hasOp {
		opStr, ok := raw.(string)
		if !ok {
			return fmt.Errorf("when clause op must be string")
		}
		if opStr == "" {
			return fmt.Errorf("when clause op must be non-empty string")
		}
		return validateOpClause(m, opStr, reg)
	}

	if err := walkLogicalChildrenMap(m, reg); err != nil {
		return err
	}
	if hasLogicalKey(m) {
		return nil
	}
	return fmt.Errorf("when clause unknown shape (missing op/and/or/not/eval)")
}

func visitEvalName(name string, reg *Registry) error {
	if _, ok := reg.Get(name); !ok {
		return fmt.Errorf("unknown evaluator %q", name)
	}
	return nil
}

func validateConstantParams(m map[string]any) error {
	p, ok := m["params"].(map[string]any)
	if !ok {
		return fmt.Errorf("when clause eval constant requires params object")
	}
	v, ok := p["value"]
	if !ok {
		return fmt.Errorf("when clause eval constant requires params.value")
	}
	if _, ok := v.(bool); !ok {
		return fmt.Errorf("when clause eval constant params.value must be boolean, got %T", v)
	}
	return nil
}

func walkLogicalChildrenMap(m map[string]any, reg *Registry) error {
	for _, key := range []string{"and", "or"} {
		v, has := m[key]
		if !has {
			continue
		}
		arr, err := asWhenSlice(v)
		if err != nil {
			return err
		}
		for _, elt := range arr {
			if err := walkWhen(elt, reg); err != nil {
				return err
			}
		}
	}
	if v, has := m["not"]; has {
		return walkWhen(v, reg)
	}
	return nil
}

func hasLogicalKey(m map[string]any) bool {
	_, a := m["and"]
	_, o := m["or"]
	_, n := m["not"]
	return a || o || n
}

func validateOpClause(m map[string]any, op string, reg *Registry) error {
	if _, ok := knownWhenOps[op]; !ok {
		return fmt.Errorf("when clause unknown op %q", op)
	}
	pathStr, err := requirePathString(m)
	if err != nil {
		return err
	}
	if err := validateSignalPathPrefix(pathStr); err != nil {
		return err
	}

	switch op {
	case "eq", "ne", "gt", "lt", "gte", "lte", "contains":
		if _, ok := m["value"]; !ok {
			return fmt.Errorf("when clause op %q requires value", op)
		}
	case "in":
		v, ok := m["value"]
		if !ok {
			return fmt.Errorf("when clause op in requires value (array)")
		}
		if _, err := asWhenSlice(v); err != nil {
			return fmt.Errorf("when clause op in requires value to be array: %w", err)
		}
	case "exists", "not_exists":
		// path only
	case "count":
		if _, ok := m["eq"]; !ok {
			return fmt.Errorf("when clause op count requires eq")
		}
		if sub, has := m["when"]; has {
			return walkWhen(sub, reg)
		}
	case "any", "all":
		sub, ok := m["when"]
		if !ok {
			return fmt.Errorf("when clause op %s requires when", op)
		}
		return walkWhen(sub, reg)
	default:
		// unreachable if knownWhenOps matches switch
	}
	return nil
}

func requirePathString(m map[string]any) (string, error) {
	raw, ok := m["path"]
	if !ok {
		return "", fmt.Errorf("when clause missing path")
	}
	p, ok := raw.(string)
	if !ok {
		return "", fmt.Errorf("when clause path must be non-empty string")
	}
	trimmed := strings.TrimSpace(p)
	if trimmed == "" {
		return "", fmt.Errorf("when clause path must be non-empty string")
	}
	if p != trimmed {
		return "", fmt.Errorf("when clause path must not have leading or trailing whitespace")
	}
	return p, nil
}

// validateSignalPathPrefix enforces dot-path roots used by observation (git, github),
// merged state (state), and quantifier element scope ($).
func validateSignalPathPrefix(path string) error {
	if path == "$" || strings.HasPrefix(path, "$.") {
		return nil
	}
	for _, prefix := range []string{"git.", "github.", "state."} {
		if strings.HasPrefix(path, prefix) {
			return nil
		}
	}
	return fmt.Errorf("when clause path %q must start with git., github., state., or $", path)
}

func rejectEvalCombiners(m map[string]any) error {
	if _, has := m["op"]; has {
		return fmt.Errorf("when clause cannot combine eval with op")
	}
	if _, has := m["and"]; has {
		return fmt.Errorf("when clause cannot combine eval with and")
	}
	if _, has := m["or"]; has {
		return fmt.Errorf("when clause cannot combine eval with or")
	}
	if _, has := m["not"]; has {
		return fmt.Errorf("when clause cannot combine eval with not")
	}
	return nil
}

func asWhenSlice(v any) ([]any, error) {
	switch t := v.(type) {
	case []any:
		return t, nil
	default:
		return nil, fmt.Errorf("when clause expected array, got %T", v)
	}
}
