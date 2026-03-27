package knowledge

import (
	"fmt"
	"strings"

	"github.com/k-shibuki/reinguard/internal/config"
	"github.com/k-shibuki/reinguard/internal/match"
)

// FilterByQuery returns entries where at least one trigger contains q as a substring (case-insensitive).
// If q is empty after trim, returns a copy of all entries.
func FilterByQuery(entries []config.KnowledgeManifestEntry, query string) []config.KnowledgeManifestEntry {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		out := make([]config.KnowledgeManifestEntry, len(entries))
		copy(out, entries)
		return out
	}
	var out []config.KnowledgeManifestEntry
	for _, e := range entries {
		for _, t := range e.Triggers {
			if strings.Contains(strings.ToLower(t), q) {
				out = append(out, e)
				break
			}
		}
	}
	return out
}

// FilterBySignals returns entries whose when-clause matches signals (match.Eval).
// Entries with nil When are always included (invalid manifests should be caught at index/validate time). On match.Eval error, the entry is included and a warning is appended (safe-side).
func FilterBySignals(entries []config.KnowledgeManifestEntry, signals map[string]any) (included []config.KnowledgeManifestEntry, warnings []string) {
	for _, e := range entries {
		if e.When == nil {
			included = append(included, e)
			continue
		}
		ok, err := match.Eval(e.When, signals)
		if err != nil {
			included = append(included, e)
			warnings = append(warnings, fmt.Sprintf("knowledge entry %q when evaluation error: %v", e.ID, err))
			continue
		}
		if ok {
			included = append(included, e)
		}
	}
	return included, warnings
}

// FilterUnion applies signal-based filtering when useSignalFilter is true, then combines with query filtering using OR (union by entry id).
// When useSignalFilter is false (no --observation-file for pack), all entries pass the signal branch (D3).
// Empty query skips the query branch for union purposes.
func FilterUnion(entries []config.KnowledgeManifestEntry, signals map[string]any, useSignalFilter bool, query string) (result []config.KnowledgeManifestEntry, warnings []string) {
	q := strings.TrimSpace(query)
	if !useSignalFilter && q == "" {
		out := make([]config.KnowledgeManifestEntry, len(entries))
		copy(out, entries)
		return out, nil
	}
	if !useSignalFilter {
		return FilterByQuery(entries, query), nil
	}
	signalSet, w := FilterBySignals(entries, signals)
	if q == "" {
		return signalSet, w
	}
	querySet := FilterByQuery(entries, query)
	return unionByEntryID(signalSet, querySet), w
}

func unionByEntryID(a, b []config.KnowledgeManifestEntry) []config.KnowledgeManifestEntry {
	seen := make(map[string]struct{})
	var out []config.KnowledgeManifestEntry
	for _, e := range a {
		if _, ok := seen[e.ID]; ok {
			continue
		}
		seen[e.ID] = struct{}{}
		out = append(out, e)
	}
	for _, e := range b {
		if _, ok := seen[e.ID]; ok {
			continue
		}
		seen[e.ID] = struct{}{}
		out = append(out, e)
	}
	return out
}
