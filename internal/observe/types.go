package observe

import "context"

// Diagnostic records a non-fatal observation issue.
type Diagnostic struct {
	Severity string `json:"severity"`
	Message  string `json:"message"`
	Provider string `json:"provider,omitempty"`
	Code     string `json:"code,omitempty"`
}

// Fragment is partial output from one provider.
type Fragment struct {
	Signals     map[string]any `json:"-"`
	Diagnostics []Diagnostic   `json:"-"`
	Degraded    bool           `json:"-"`
}

// Provider collects externally observable signals (ADR-0003, ADR-0005).
type Provider interface {
	ID() string
	Collect(ctx context.Context, opts Options) (Fragment, error)
}

// Options configure a collect run.
type Options struct {
	WorkDir       string
	GitHubFacet   string
	DefaultBranch string
	ProviderIDs   []string
	Serial        bool
}
