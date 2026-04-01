package observe

import (
	"context"
	"errors"
	"fmt"
	"sync"

	"github.com/k-shibuki/reinguard/internal/config"
	"github.com/k-shibuki/reinguard/internal/observe/github/issues"
)

// Engine runs configured providers concurrently or serially and merges fragments into a
// provider-keyed signal map with aggregated diagnostics (see package doc).
type Engine struct {
	Providers map[string]Provider
}

// NewEngine builds an engine with default built-in providers (git + github), no per-config options.
func NewEngine() *Engine {
	e, err := NewEngineFromConfig([]config.ProviderSpec{
		{ID: "git", Enabled: true},
		{ID: "github", Enabled: true},
	})
	if err != nil {
		panic("observe: NewEngine: " + err.Error())
	}
	return e
}

// NewEngineFromConfig builds an engine from provider specs using the default registry (ADR-0009).
func NewEngineFromConfig(specs []config.ProviderSpec) (*Engine, error) {
	reg := DefaultRegistry()
	providers, err := reg.Build(specs)
	if err != nil {
		return nil, err
	}
	return &Engine{Providers: providers}, nil
}

// Collect runs enabled providers from root (or opts.ProviderIDs when set) and returns merged
// signals, diagnostics, degraded, and an error only for invalid inputs or missing providers.
func (e *Engine) Collect(ctx context.Context, root *config.Root, opts Options) (map[string]any, []Diagnostic, bool, error) {
	if e == nil {
		return nil, nil, false, fmt.Errorf("observe: nil engine")
	}
	if root == nil {
		return nil, nil, false, fmt.Errorf("observe: nil config root")
	}
	enabled := root.EnabledProviderIDs()
	if len(opts.ProviderIDs) > 0 {
		enabled = append([]string(nil), opts.ProviderIDs...)
	}
	if len(enabled) == 0 {
		return map[string]any{}, nil, false, nil
	}

	opts.DefaultBranch = root.DefaultBranch

	type res struct {
		err  error
		id   string
		frag Fragment
	}

	results := make([]res, len(enabled))
	if opts.Serial {
		for i, id := range enabled {
			p, ok := e.Providers[id]
			if !ok {
				results[i] = res{id: id, err: fmt.Errorf("unknown provider %q", id)}
				continue
			}
			frag, err := p.Collect(ctx, opts)
			results[i] = res{id: id, frag: frag, err: err}
		}
	} else {
		var wg sync.WaitGroup
		for i, id := range enabled {
			i, id := i, id
			p, ok := e.Providers[id]
			if !ok {
				results[i] = res{id: id, err: fmt.Errorf("unknown provider %q", id)}
				continue
			}
			wg.Add(1)
			go func() {
				defer wg.Done()
				frag, err := p.Collect(ctx, opts)
				results[i] = res{id: id, frag: frag, err: err}
			}()
		}
		wg.Wait()
	}

	signals := make(map[string]any)
	var diags []Diagnostic
	degraded := false

	for _, r := range results {
		if r.err != nil {
			if errors.Is(r.err, issues.ErrFatalObservation) {
				return nil, diags, degraded, r.err
			}
			degraded = true
			diags = append(diags, Diagnostic{
				Severity: "error",
				Message:  r.err.Error(),
				Provider: r.id,
				Code:     "provider_failed",
			})
			continue
		}
		if r.frag.Degraded {
			degraded = true
			diags = append(diags, Diagnostic{
				Severity: "warning",
				Message:  "provider returned degraded partial data",
				Provider: r.id,
				Code:     "provider_degraded",
			})
		}
		diags = append(diags, r.frag.Diagnostics...)
		if r.frag.Signals != nil {
			signals[r.id] = r.frag.Signals
		}
	}

	return signals, diags, degraded, nil
}
