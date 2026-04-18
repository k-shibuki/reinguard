package observation

import (
	"testing"

	"github.com/k-shibuki/reinguard/internal/observe"
	"github.com/k-shibuki/reinguard/pkg/schema"
)

func TestDocument_andDegradedSet(t *testing.T) {
	t.Parallel()
	// Given: signals, provider_failed diagnostic, degraded flag
	diags := []observe.Diagnostic{{Severity: "error", Message: "x", Provider: "git", Code: "provider_failed"}}
	// When: Document and DegradedSet are built
	doc := Document(map[string]any{"a": 1}, diags, true, nil)
	ds := DegradedSet(diags, true)
	// Then: degraded true and git in set
	if doc["degraded"] != true {
		t.Fatal()
	}
	if doc["schema_version"] != schema.CurrentSchemaVersion {
		t.Fatal(doc["schema_version"])
	}
	if _, ok := ds["git"]; !ok {
		t.Fatal()
	}
}

func TestDocument_noDiagnosticsNoMeta(t *testing.T) {
	t.Parallel()
	// Given: no diagnostics
	// When: Document is built
	doc := Document(map[string]any{"k": 2}, nil, false, nil)
	// Then: no diagnostics key required in map — implementation omits empty diagnostics
	if _, ok := doc["diagnostics"]; ok {
		t.Fatal("expected no diagnostics key when empty")
	}
	if _, ok := doc["meta"]; ok {
		t.Fatal("expected no meta when no degraded sources")
	}
}

func TestDegradedSet_providerDegradedCode(t *testing.T) {
	t.Parallel()
	// Given: diagnostic with provider_degraded code
	diags := []observe.Diagnostic{{Provider: "github", Code: "provider_degraded", Severity: "warning", Message: "m"}}
	// When: DegradedSet runs
	ds := DegradedSet(diags, false)
	// Then: provider in set
	if _, ok := ds["github"]; !ok {
		t.Fatal()
	}
}

func TestDegradedSet_fallbackErrorSeverity(t *testing.T) {
	t.Parallel()
	// Given: degraded true but no provider_failed code — fallback uses error severity
	diags := []observe.Diagnostic{{Provider: "x", Code: "other", Severity: "error", Message: "m"}}
	// When: DegradedSet runs
	ds := DegradedSet(diags, true)
	// Then: provider included via error-severity fallback
	if _, ok := ds["x"]; !ok {
		t.Fatal()
	}
}

func TestDegradedSet_multipleProvidersDeduped(t *testing.T) {
	t.Parallel()
	// Given: duplicate diagnostics for same provider/code
	diags := []observe.Diagnostic{
		{Provider: "git", Code: "provider_failed", Severity: "error", Message: "a"},
		{Provider: "git", Code: "provider_failed", Severity: "error", Message: "b"},
	}
	// When: DegradedSet runs
	ds := DegradedSet(diags, false)
	// Then: single provider entry (deduped)
	if len(ds) != 1 {
		t.Fatal(ds)
	}
}

func TestDocument_withDiagnosticsAndMeta(t *testing.T) {
	t.Parallel()
	// Given: two diagnostics from different providers with qualifying codes
	diags := []observe.Diagnostic{
		{Severity: "warn", Message: "m1", Provider: "git", Code: "provider_failed"},
		{Severity: "warn", Message: "m2", Provider: "github", Code: "provider_degraded"},
	}
	// When: Document is built with degraded=true
	doc := Document(map[string]any{"x": 1}, diags, true, nil)
	// Then: diagnostics array has 2 entries
	raw, ok := doc["diagnostics"].([]any)
	if !ok || len(raw) != 2 {
		t.Fatalf("diagnostics: %v", doc["diagnostics"])
	}
	// Then: meta contains degraded_sources with both providers
	meta, ok := doc["meta"].(map[string]any)
	if !ok {
		t.Fatal("expected meta")
	}
	srcs, ok := meta["degraded_sources"].([]any)
	if !ok || len(srcs) != 2 {
		t.Fatalf("degraded_sources: %v", meta["degraded_sources"])
	}
	found := map[string]bool{}
	for _, s := range srcs {
		if str, ok := s.(string); ok {
			found[str] = true
		}
	}
	if !found["git"] || !found["github"] {
		t.Fatalf("expected git and github in degraded_sources, got: %v", srcs)
	}
}

func TestDocument_metaContract(t *testing.T) {
	t.Parallel()
	tests := map[string]func(*testing.T){
		"preserves view": func(t *testing.T) {
			t.Helper()
			doc := Document(map[string]any{"x": 1}, nil, false, map[string]any{"view": "summary"})
			meta, ok := doc["meta"].(map[string]any)
			if !ok {
				t.Fatal("expected meta")
			}
			if got := meta["view"]; got != "summary" {
				t.Fatalf("view=%v", got)
			}
		},
		"removes stale degraded_sources when recomputation is empty": func(t *testing.T) {
			t.Helper()
			doc := Document(
				map[string]any{"x": 1},
				nil,
				false,
				map[string]any{"view": "summary", "degraded_sources": []any{"stale"}},
			)
			meta, ok := doc["meta"].(map[string]any)
			if !ok {
				t.Fatal("expected meta")
			}
			if got := meta["view"]; got != "summary" {
				t.Fatalf("view=%v", got)
			}
			if _, exists := meta["degraded_sources"]; exists {
				t.Fatalf("unexpected degraded_sources=%v", meta["degraded_sources"])
			}
		},
		"overwrites stale degraded_sources with recomputed sources": func(t *testing.T) {
			t.Helper()
			doc := Document(
				map[string]any{"x": 1},
				[]observe.Diagnostic{
					{Provider: "git", Code: "provider_failed", Severity: "error", Message: "m"},
				},
				true,
				map[string]any{"view": "summary", "degraded_sources": []any{"stale"}},
			)
			meta, ok := doc["meta"].(map[string]any)
			if !ok {
				t.Fatal("expected meta")
			}
			if got := meta["view"]; got != "summary" {
				t.Fatalf("view=%v", got)
			}
			srcs, ok := meta["degraded_sources"].([]any)
			if !ok || len(srcs) != 1 {
				t.Fatalf("degraded_sources=%v", meta["degraded_sources"])
			}
			if srcs[0] != "git" {
				t.Fatalf("degraded_sources[0]=%v, want=%q", srcs[0], "git")
			}
		},
	}
	for name, run := range tests {
		name := name
		run := run
		t.Run(name, func(t *testing.T) {
			t.Parallel()
			run(t)
		})
	}
}
