package evaluator

import (
	"fmt"
)

// ValidateWhen checks that every eval: reference in when is registered in reg.
func ValidateWhen(when any, reg *Registry) error {
	if reg == nil {
		reg = DefaultRegistry()
	}
	return walkWhenEval(when, func(name string) error {
		if _, ok := reg.Get(name); !ok {
			return fmt.Errorf("unknown evaluator %q", name)
		}
		return nil
	})
}

func walkWhenEval(when any, visit func(name string) error) error {
	if when == nil {
		return nil
	}
	switch t := when.(type) {
	case map[string]any:
		return walkWhenMap(t, visit)
	case []any:
		for _, elt := range t {
			if err := walkWhenEval(elt, visit); err != nil {
				return err
			}
		}
		return nil
	default:
		return nil
	}
}

func walkWhenMap(m map[string]any, visit func(name string) error) error {
	name, ok := m["eval"].(string)
	if ok && name != "" {
		return finishEvalClause(m, name, visit)
	}
	if err := walkLogicalChildren(m, visit); err != nil {
		return err
	}
	return walkOpNestedWhen(m, visit)
}

func finishEvalClause(m map[string]any, name string, visit func(name string) error) error {
	if err := visit(name); err != nil {
		return err
	}
	return rejectEvalCombiners(m)
}

func rejectEvalCombiners(m map[string]any) error {
	if _, has := m["op"].(string); has {
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

func walkLogicalChildren(m map[string]any, visit func(name string) error) error {
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
			if err := walkWhenEval(elt, visit); err != nil {
				return err
			}
		}
	}
	if v, has := m["not"]; has {
		return walkWhenEval(v, visit)
	}
	return nil
}

func walkOpNestedWhen(m map[string]any, visit func(name string) error) error {
	op, ok := m["op"].(string)
	if !ok {
		return nil
	}
	switch op {
	case "count", "any", "all":
		if sub, has := m["when"]; has {
			return walkWhenEval(sub, visit)
		}
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
