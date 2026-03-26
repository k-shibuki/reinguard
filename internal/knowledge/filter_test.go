package knowledge

import (
	"testing"

	"github.com/k-shibuki/reinguard/internal/config"
)

func TestFilterByQuery_emptyQueryReturnsAll(t *testing.T) {
	t.Parallel()
	// Given: entries and whitespace-only query
	entries := []config.KnowledgeManifestEntry{
		{ID: "a", Path: "p", Description: "d", Triggers: []string{"x"}},
	}
	// When: FilterByQuery runs
	out := FilterByQuery(entries, "  ")
	// Then: all entries returned
	if len(out) != 1 {
		t.Fatalf("%v", out)
	}
}

func TestFilterByQuery_match(t *testing.T) {
	t.Parallel()
	// Given: entries whose triggers include a substring of the query
	entries := []config.KnowledgeManifestEntry{
		{ID: "a", Path: "p1", Description: "d", Triggers: []string{"alpha", "beta"}},
		{ID: "b", Path: "p2", Description: "d", Triggers: []string{"gamma"}},
	}
	// When: FilterByQuery runs with "alp"
	out := FilterByQuery(entries, "alp")
	// Then: only matching entry "a"
	if len(out) != 1 || out[0].ID != "a" {
		t.Fatalf("%v", out)
	}
}

func TestFilterByQuery_caseInsensitive(t *testing.T) {
	t.Parallel()
	// Given: trigger with mixed case
	entries := []config.KnowledgeManifestEntry{
		{ID: "a", Path: "p", Description: "d", Triggers: []string{"JSON"}},
	}
	// When: query is lower case
	out := FilterByQuery(entries, "json")
	// Then: match
	if len(out) != 1 {
		t.Fatalf("%v", out)
	}
}

func TestFilterByQuery_noMatchReturnsEmpty(t *testing.T) {
	t.Parallel()
	// Given: query that matches no trigger
	entries := []config.KnowledgeManifestEntry{
		{ID: "a", Path: "p1", Description: "d", Triggers: []string{"alpha"}},
	}
	// When: FilterByQuery runs
	out := FilterByQuery(entries, "zzz")
	// Then: empty slice
	if len(out) != 0 {
		t.Fatalf("%v", out)
	}
}
