package evaluator

import "fmt"

// constantEvaluator implements eval: constant — returns params["value"] as a bool (YAML/JSON boolean).
type constantEvaluator struct{}

func (constantEvaluator) Name() string { return "constant" }

func (constantEvaluator) Eval(_ map[string]any, params map[string]any) (any, error) {
	if params == nil {
		return nil, fmt.Errorf("constant: missing params.value")
	}
	v, ok := params["value"]
	if !ok {
		return nil, fmt.Errorf("constant: missing params.value")
	}
	b, ok := v.(bool)
	if !ok {
		return nil, fmt.Errorf("constant: params.value must be boolean, got %T", v)
	}
	return b, nil
}
