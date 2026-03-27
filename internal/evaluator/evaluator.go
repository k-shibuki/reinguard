// Package evaluator registers named evaluators for ADR-0002 match-layer when-clauses.
// Guard built-ins (OK/Reason) live in package guard; this registry is only for
// { eval: "<name>", params: {...} } nodes inside when expressions.
package evaluator

import (
	"fmt"
	"sort"
)

// Evaluator is a compiled named evaluator invoked from match rules (ADR-0002).
type Evaluator interface {
	Name() string
	Eval(signals map[string]any, params map[string]any) (any, error)
}

// Registry maps evaluator names to implementations.
type Registry struct {
	byName map[string]Evaluator
}

var defaultRegistry = newBuiltinRegistry()

func newBuiltinRegistry() *Registry {
	r := NewRegistry()
	if err := r.Register(constantEvaluator{}); err != nil {
		panic("evaluator: builtin registration: " + err.Error())
	}
	return r
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{byName: make(map[string]Evaluator)}
}

// DefaultRegistry returns the process-wide registry with standard built-ins.
func DefaultRegistry() *Registry {
	return defaultRegistry
}

// Register adds an evaluator. Names must be unique.
func (r *Registry) Register(e Evaluator) error {
	if r == nil {
		return fmt.Errorf("evaluator: nil registry")
	}
	if e == nil {
		return fmt.Errorf("evaluator: nil evaluator")
	}
	if r.byName == nil {
		r.byName = make(map[string]Evaluator)
	}
	name := e.Name()
	if name == "" {
		return fmt.Errorf("evaluator: empty name")
	}
	if _, exists := r.byName[name]; exists {
		return fmt.Errorf("evaluator: %q already registered", name)
	}
	r.byName[name] = e
	return nil
}

// Get returns the evaluator for name, if registered.
func (r *Registry) Get(name string) (Evaluator, bool) {
	if r == nil {
		return nil, false
	}
	e, ok := r.byName[name]
	return e, ok
}

// ListRegistered returns registered names in sorted order (stable design-metric / review surface).
func (r *Registry) ListRegistered() []string {
	if r == nil || len(r.byName) == 0 {
		return nil
	}
	out := make([]string, 0, len(r.byName))
	for n := range r.byName {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}
