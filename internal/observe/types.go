package observe

import "context"

// View describes the requested observation depth for CLI-driven collection.
type View string

// Supported observation views.
const (
	ViewSummary View = "summary"
	ViewInbox   View = "inbox"
	ViewFull    View = "full"
)

// Valid reports whether v is one of the supported observation views.
func (v View) Valid() bool {
	switch v {
	case ViewSummary, ViewInbox, ViewFull:
		return true
	default:
		return false
	}
}

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

// Scope selects an explicit GitHub PR/branch target for one collect run.
// Zero values mean "infer from the current working directory". When both fields are set,
// PRNumber takes precedence for PR-scoped GitHub facets.
type Scope struct {
	Branch   string
	PRNumber int
}

// Options configure a collect run: working directory, optional GitHub facet filter, default
// branch from config, optional provider ID restriction, explicit PR/branch scope, and serial
// vs parallel execution. View defaults are selected by callers; the zero value is not valid on
// its own.
type Options struct {
	ProviderIDs   []string
	WorkDir       string
	GitHubFacet   string
	DefaultBranch string
	View          View
	Scope         Scope
	Serial        bool
}
