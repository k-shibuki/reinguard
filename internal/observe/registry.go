package observe

import (
	"fmt"
	"strings"

	"github.com/k-shibuki/reinguard/internal/config"
)

// ProviderFactory constructs a Provider from per-spec options (ADR-0009).
type ProviderFactory func(opts map[string]any) (Provider, error)

// ProviderRegistry maps provider ids to factories.
type ProviderRegistry struct {
	factories map[string]ProviderFactory
}

// NewRegistry returns an empty registry.
func NewRegistry() *ProviderRegistry {
	return &ProviderRegistry{factories: make(map[string]ProviderFactory)}
}

// Register binds id to factory. Duplicate ids return an error.
func (r *ProviderRegistry) Register(id string, factory ProviderFactory) error {
	if r == nil {
		return fmt.Errorf("observe: nil registry")
	}
	id = strings.TrimSpace(id)
	if id == "" {
		return fmt.Errorf("observe: register provider: empty id")
	}
	if factory == nil {
		return fmt.Errorf("observe: register provider %q: nil factory", id)
	}
	if _, exists := r.factories[id]; exists {
		return fmt.Errorf("observe: register provider %q: duplicate", id)
	}
	r.factories[id] = factory
	return nil
}

// Build constructs providers for each enabled spec. Disabled specs are skipped.
// Options are shallow-copied before passing to the factory.
func (r *ProviderRegistry) Build(specs []config.ProviderSpec) (map[string]Provider, error) {
	if r == nil {
		return nil, fmt.Errorf("observe: nil registry")
	}
	out := make(map[string]Provider)
	for i, spec := range specs {
		if !spec.Enabled {
			continue
		}
		id := strings.TrimSpace(spec.ID)
		if id == "" {
			return nil, fmt.Errorf("observe: providers[%d]: empty id", i)
		}
		if _, dup := out[id]; dup {
			return nil, fmt.Errorf("observe: duplicate provider id %q", id)
		}
		f, ok := r.factories[id]
		if !ok {
			return nil, fmt.Errorf("observe: unknown provider id %q", id)
		}
		opts := shallowCopyMap(spec.Options)
		p, err := f(opts)
		if err != nil {
			return nil, fmt.Errorf("observe: build provider %q: %w", id, err)
		}
		if p == nil {
			return nil, fmt.Errorf("observe: provider %q factory returned nil", id)
		}
		if got := strings.TrimSpace(p.ID()); got != id {
			return nil, fmt.Errorf("observe: provider factory for %q returned provider id %q", id, got)
		}
		out[id] = p
	}
	return out, nil
}

// DefaultRegistry registers built-in git and github factories.
func DefaultRegistry() *ProviderRegistry {
	r := NewRegistry()
	if err := r.Register("git", GitProviderFactory); err != nil {
		panic("observe: DefaultRegistry: " + err.Error())
	}
	if err := r.Register("github", GitHubProviderFactory); err != nil {
		panic("observe: DefaultRegistry: " + err.Error())
	}
	return r
}

func shallowCopyMap(m map[string]any) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	out := make(map[string]any, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}
