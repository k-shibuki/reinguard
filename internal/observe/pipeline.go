package observe

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/k-shibuki/reinguard/internal/config"
)

// LoadSignalsOptions configures LoadSignalsFileOrCollect (file vs live collect).
type LoadSignalsOptions struct {
	// ObservationPath is a resolved filesystem path to observation JSON.
	// When empty, Collect runs against the repository.
	ObservationPath string
	WorkDir         string
	Serial          bool
}

// LoadSignalsFileOrCollect reads observation JSON from ObservationPath when set;
// otherwise builds an engine from root.Providers via NewEngineFromConfig and
// runs Collect (ADR-0003 pull-based collect). An empty provider list yields an
// engine with no providers, not a legacy fixed default.
func LoadSignalsFileOrCollect(ctx context.Context, root *config.Root, opts LoadSignalsOptions) (map[string]any, []Diagnostic, bool, error) {
	if opts.ObservationPath != "" {
		data, err := os.ReadFile(opts.ObservationPath)
		if err != nil {
			return nil, nil, false, err
		}
		return ParseObservationJSON(data)
	}
	if root == nil {
		return nil, nil, false, fmt.Errorf("observe: nil config root")
	}
	engine, err := NewEngineFromConfig(root.Providers)
	if err != nil {
		return nil, nil, false, err
	}
	return engine.Collect(ctx, root, Options{WorkDir: opts.WorkDir, Serial: opts.Serial})
}

// ParseObservationJSON decodes a saved observation document (signals, diagnostics, degraded).
func ParseObservationJSON(data []byte) (signals map[string]any, diags []Diagnostic, degraded bool, err error) {
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, nil, false, err
	}
	rawSignals, ok := doc["signals"]
	if !ok {
		return nil, nil, false, fmt.Errorf("observation JSON must include object field %q", "signals")
	}
	signals, ok = rawSignals.(map[string]any)
	if !ok {
		return nil, nil, false, fmt.Errorf("observation JSON field %q must be an object", "signals")
	}
	degraded, _ = doc["degraded"].(bool)
	if raw, ok := doc["diagnostics"].([]any); ok {
		for _, r := range raw {
			if m, ok := r.(map[string]any); ok {
				diags = append(diags, Diagnostic{
					Severity: stringField(m, "severity"),
					Message:  stringField(m, "message"),
					Provider: stringField(m, "provider"),
					Code:     stringField(m, "code"),
				})
			}
		}
	}
	return signals, diags, degraded, nil
}

func stringField(m map[string]any, k string) string {
	if v, ok := m[k].(string); ok {
		return v
	}
	return ""
}
