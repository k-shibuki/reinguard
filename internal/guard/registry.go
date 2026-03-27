package guard

import "fmt"

// Registry maps guard IDs to built-in implementations.
type Registry struct {
	byID map[string]Guard
}

// defaultRegistry is the process-wide registry initialized with standard built-ins.
var defaultRegistry = newBuiltinRegistry()

func newBuiltinRegistry() *Registry {
	r := NewRegistry()
	if err := r.Register(mergeReadinessGuard{}); err != nil {
		panic("guard: builtin registration: " + err.Error())
	}
	return r
}

// NewRegistry returns an empty registry.
func NewRegistry() *Registry {
	return &Registry{byID: make(map[string]Guard)}
}

// DefaultRegistry returns the process-wide registry with standard built-ins (merge-readiness).
func DefaultRegistry() *Registry {
	return defaultRegistry
}

// Register adds a built-in guard. IDs must be unique.
func (r *Registry) Register(g Guard) error {
	id := g.ID()
	if _, exists := r.byID[id]; exists {
		return fmt.Errorf("guard: %q already registered", id)
	}
	r.byID[id] = g
	return nil
}

// Lookup returns the built-in for id, if registered.
func (r *Registry) Lookup(id string) (Guard, bool) {
	g, ok := r.byID[id]
	return g, ok
}
