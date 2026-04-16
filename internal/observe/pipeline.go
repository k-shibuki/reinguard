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
	// Branch observes GitHub PR linkage for a specific branch.
	// When empty, live collection uses the checked-out branch.
	Branch string
	// View selects the observation output view (summary, inbox, full).
	View View
	// PRNumber observes GitHub PR-scoped facets for a specific pull request.
	// When zero, live collection does not force a PR number.
	PRNumber int
	Serial   bool
}

// ParsedObservation captures the supported observation document fields.
type ParsedObservation struct {
	Signals map[string]any
	Meta    map[string]any

	Diagnostics []Diagnostic
	Degraded    bool
}

// LoadSignalsFileOrCollect reads observation JSON from ObservationPath when set (signals,
// diagnostics, degraded per observation document shape); otherwise builds an engine from
// root.Providers via NewEngineFromConfig and runs Collect (ADR-0003). An empty provider list
// yields an engine with no providers, not a legacy fixed default. Returns file/JSON errors
// on read path; nil root with no observation file returns an error.
func LoadSignalsFileOrCollect(ctx context.Context, root *config.Root, opts LoadSignalsOptions) (map[string]any, []Diagnostic, bool, error) {
	if opts.ObservationPath != "" {
		doc, err := LoadObservationFile(opts.ObservationPath)
		if err != nil {
			return nil, nil, false, err
		}
		return doc.Signals, doc.Diagnostics, doc.Degraded, nil
	}
	if root == nil {
		return nil, nil, false, fmt.Errorf("observe: nil config root")
	}
	engine, err := NewEngineFromConfig(root.Providers)
	if err != nil {
		return nil, nil, false, err
	}
	return engine.Collect(ctx, root, Options{
		WorkDir: opts.WorkDir,
		Scope:   Scope{Branch: opts.Branch, PRNumber: opts.PRNumber},
		View:    opts.View,
		Serial:  opts.Serial,
	})
}

// LoadObservationFile reads and parses a saved observation document from disk.
func LoadObservationFile(path string) (ParsedObservation, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return ParsedObservation{}, err
	}
	return ParseObservationDocument(data)
}

// ParseObservationDocument decodes a saved observation document including optional meta.
func ParseObservationDocument(data []byte) (ParsedObservation, error) {
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return ParsedObservation{}, err
	}
	rawSignals, ok := doc["signals"]
	if !ok {
		return ParsedObservation{}, fmt.Errorf("observation JSON must include object field %q", "signals")
	}
	signals, ok := rawSignals.(map[string]any)
	if !ok {
		return ParsedObservation{}, fmt.Errorf("observation JSON field %q must be an object", "signals")
	}
	out := ParsedObservation{
		Signals: signals,
	}
	out.Degraded, _ = doc["degraded"].(bool)
	if rawMeta, ok := doc["meta"].(map[string]any); ok {
		out.Meta = rawMeta
	}
	if raw, ok := doc["diagnostics"].([]any); ok {
		for _, r := range raw {
			if m, ok := r.(map[string]any); ok {
				out.Diagnostics = append(out.Diagnostics, Diagnostic{
					Severity: stringField(m, "severity"),
					Message:  stringField(m, "message"),
					Provider: stringField(m, "provider"),
					Code:     stringField(m, "code"),
				})
			}
		}
	}
	return out, nil
}

// ParseObservationJSON decodes a saved observation document (signals, diagnostics, degraded).
func ParseObservationJSON(data []byte) (signals map[string]any, diags []Diagnostic, degraded bool, err error) {
	doc, err := ParseObservationDocument(data)
	if err != nil {
		return nil, nil, false, err
	}
	return doc.Signals, doc.Diagnostics, doc.Degraded, nil
}

func stringField(m map[string]any, k string) string {
	if v, ok := m[k].(string); ok {
		return v
	}
	return ""
}
