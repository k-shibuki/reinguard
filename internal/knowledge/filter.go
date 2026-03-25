package knowledge

import (
	"strings"

	"github.com/k-shibuki/reinguard/internal/config"
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
