package knowledge

import (
	"testing"

	"github.com/k-shibuki/reinguard/internal/config"
)

func TestFilterByQuery_emptyQueryReturnsAll(t *testing.T) {
	t.Parallel()
	entries := []config.KnowledgeManifestEntry{
		{ID: "a", Path: "p", Description: "d", Triggers: []string{"x"}},
	}
	out := FilterByQuery(entries, "  ")
	if len(out) != 1 {
		t.Fatalf("%v", out)
	}
}

func TestFilterByQuery_match(t *testing.T) {
	t.Parallel()
	entries := []config.KnowledgeManifestEntry{
		{ID: "a", Path: "p1", Description: "d", Triggers: []string{"alpha", "beta"}},
		{ID: "b", Path: "p2", Description: "d", Triggers: []string{"gamma"}},
	}
	out := FilterByQuery(entries, "alp")
	if len(out) != 1 || out[0].ID != "a" {
		t.Fatalf("%v", out)
	}
}

func TestFilterByQuery_caseInsensitive(t *testing.T) {
	t.Parallel()
	entries := []config.KnowledgeManifestEntry{
		{ID: "a", Path: "p", Description: "d", Triggers: []string{"JSON"}},
	}
	out := FilterByQuery(entries, "json")
	if len(out) != 1 {
		t.Fatalf("%v", out)
	}
}
