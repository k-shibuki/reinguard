package guard

// Registry maps guard IDs to built-in implementations.
type Registry struct {
	byID map[string]Guard
}

var defaultRegistry = newBuiltinRegistry()

func newBuiltinRegistry() *Registry {
	r := NewRegistry()
	r.Register(mergeReadinessGuard{})
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
func (r *Registry) Register(g Guard) {
	r.byID[g.ID()] = g
}

// Lookup returns the built-in for id, if registered.
func (r *Registry) Lookup(id string) (Guard, bool) {
	g, ok := r.byID[id]
	return g, ok
}
