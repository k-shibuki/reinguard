package observe

import (
	"context"
	"fmt"
	"sync"

	"github.com/k-shibuki/reinguard/internal/config"
)

// Engine runs configured providers and merges into an observation-shaped map.
type Engine struct {
	Providers map[string]Provider
}

// NewEngine builds an engine from the default registry.
func NewEngine() *Engine {
	return &Engine{Providers: defaultRegistry()}
}

// Collect runs enabled providers from root config and returns merged signals + diagnostics.
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
