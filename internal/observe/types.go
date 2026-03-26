package observe

import "context"

// Diagnostic records a non-fatal observation issue (provider failure, warning, or fragment-level detail).
type Diagnostic struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Provider string `json:"provider,omitempty"`
	Code     string `json:"code,omitempty"`
}

// Fragment is partial output from one provider before merge into the engine result.
type Fragment struct {
	Signals     map[string]any `json:"-"`
	Diagnostics []Diagnostic   `json:"-"`
	Degraded    bool           `json:"-"`
}

// Provider collects externally observable signals into a Fragment (ADR-0003, ADR-0005).
// Implementations return errors for unrecoverable failures; those are converted to diagnostics
// by the engine, which may still merge other providers.
type Provider interface {
	ID() string
	Collect(ctx context.Context, opts Options) (Fragment, error)
}

// Options configure a collect run: working directory, optional GitHub facet filter, default
// branch from config, optional provider ID restriction, and serial vs parallel execution.
type Options struct {
	WorkDir       string
	GitHubFacet   string
	DefaultBranch string
	ProviderIDs   []string
	Serial        bool
}
