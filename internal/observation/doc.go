// Package observation builds validated observation document payloads from provider signals
// and diagnostics (schema_version, signals, degraded, optional meta).
package observation

import (
	"github.com/k-shibuki/reinguard/internal/observe"
	"github.com/k-shibuki/reinguard/pkg/schema"
)

// Document builds an observation document map (validated against observation schema by caller).
// Caller-provided meta is copied first; reserved keys such as degraded_sources are overwritten
// by the computed observation metadata.
func Document(signals map[string]any, diags []observe.Diagnostic, degraded bool, meta map[string]any) map[string]any {
	srcs := degradedSources(diags, degraded)
	mergedMeta := map[string]any{}
	for k, v := range meta {
		mergedMeta[k] = v
	}
	if len(srcs) > 0 {
		list := make([]any, len(srcs))
		for i, s := range srcs {
			list[i] = s
		}
		mergedMeta["degraded_sources"] = list
	}
	doc := map[string]any{
		"schema_version": schema.CurrentSchemaVersion,
		"signals":        signals,
		"degraded":       degraded,
	}
	if len(diags) > 0 {
		ds := make([]any, len(diags))
		for i := range diags {
			d := diags[i]
			ds[i] = map[string]any{
				"severity": d.Severity,
				"message":  d.Message,
				"provider": d.Provider,
				"code":     d.Code,
			}
		}
		doc["diagnostics"] = ds
	}
	if len(mergedMeta) > 0 {
		doc["meta"] = mergedMeta
	}
	return doc
}

func degradedSources(diags []observe.Diagnostic, degraded bool) []string {
	seen := map[string]struct{}{}
	var out []string
	for _, d := range diags {
		if d.Provider == "" {
			continue
		}
		if d.Code == "provider_failed" || d.Code == "provider_degraded" {
			if _, ok := seen[d.Provider]; !ok {
				seen[d.Provider] = struct{}{}
				out = append(out, d.Provider)
			}
		}
	}
	if degraded && len(out) == 0 {
		// Fallback: any error-level diagnostic ties to provider.
		for _, d := range diags {
			if d.Provider != "" && d.Severity == "error" {
				if _, ok := seen[d.Provider]; !ok {
					seen[d.Provider] = struct{}{}
					out = append(out, d.Provider)
				}
			}
		}
	}
	return out
}

// DegradedSet returns a set of source names for resolve.DependsOn suppression.
func DegradedSet(diags []observe.Diagnostic, degraded bool) map[string]struct{} {
	m := map[string]struct{}{}
	for _, s := range degradedSources(diags, degraded) {
		m[s] = struct{}{}
	}
	return m
}
