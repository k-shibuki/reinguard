package knowledge

import (
	"strings"
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

func TestFilterBySignals_noWhenAlwaysIncluded(t *testing.T) {
	t.Parallel()
	entries := []config.KnowledgeManifestEntry{
		{ID: "a", Path: "p", Description: "d", Triggers: []string{"t"}},
	}
	sig := map[string]any{"git": map[string]any{"branch": "main"}}
	out, w := FilterBySignals(entries, sig)
	if len(out) != 1 || len(w) != 0 {
		t.Fatalf("out=%v w=%v", out, w)
	}
}

func TestFilterBySignals_whenMatch(t *testing.T) {
	t.Parallel()
	entries := []config.KnowledgeManifestEntry{
		{
			ID: "a", Path: "p", Description: "d", Triggers: []string{"t"},
			When: map[string]any{
				"op": "eq", "path": "git.branch", "value": "main",
			},
		},
		{
			ID: "b", Path: "p2", Description: "d", Triggers: []string{"t"},
			When: map[string]any{
				"op": "eq", "path": "git.branch", "value": "other",
			},
		},
	}
	sig := map[string]any{"git": map[string]any{"branch": "main"}}
	out, w := FilterBySignals(entries, sig)
	if len(w) != 0 {
		t.Fatalf("warnings: %v", w)
	}
	if len(out) != 1 || out[0].ID != "a" {
		t.Fatalf("got %+v", out)
	}
}

func TestFilterBySignals_evalErrorIncludesWithWarning(t *testing.T) {
	t.Parallel()
	entries := []config.KnowledgeManifestEntry{
		{
			ID: "bad", Path: "p", Description: "d", Triggers: []string{"t"},
			When: map[string]any{
				"op": "no_such_op", "path": "git.branch", "value": "x",
			},
		},
	}
	sig := map[string]any{"git": map[string]any{"branch": "main"}}
	out, w := FilterBySignals(entries, sig)
	if len(out) != 1 || out[0].ID != "bad" {
		t.Fatalf("out=%v", out)
	}
	if len(w) != 1 || !strings.Contains(w[0], "bad") {
		t.Fatalf("warnings=%v", w)
	}
}

func TestFilterUnion_noSignalsNoQueryReturnsAll(t *testing.T) {
	t.Parallel()
	entries := []config.KnowledgeManifestEntry{
		{ID: "a", Path: "p", Description: "d", Triggers: []string{"t"}},
	}
	out, w := FilterUnion(entries, nil, false, "")
	if len(out) != 1 || len(w) != 0 {
		t.Fatalf("out=%v w=%v", out, w)
	}
}

func TestFilterUnion_queryOnly(t *testing.T) {
	t.Parallel()
	entries := []config.KnowledgeManifestEntry{
		{ID: "a", Path: "p1", Description: "d", Triggers: []string{"apple"}},
		{ID: "b", Path: "p2", Description: "d", Triggers: []string{"beta"}},
	}
	out, w := FilterUnion(entries, nil, false, "app")
	if len(w) != 0 || len(out) != 1 || out[0].ID != "a" {
		t.Fatalf("out=%v", out)
	}
}

func TestFilterUnion_signalAndQueryOR(t *testing.T) {
	t.Parallel()
	entries := []config.KnowledgeManifestEntry{
		{
			ID: "sig", Path: "p1", Description: "d", Triggers: []string{"x"},
			When: map[string]any{"op": "eq", "path": "git.branch", "value": "main"},
		},
		{
			ID: "qonly", Path: "p2", Description: "d", Triggers: []string{"banana"},
		},
	}
	flat := map[string]any{"git": map[string]any{"branch": "main"}}
	out, w := FilterUnion(entries, flat, true, "ban")
	if len(w) != 0 {
		t.Fatalf("w=%v", w)
	}
	if len(out) != 2 {
		t.Fatalf("want union of both entries, got %+v", out)
	}
}
