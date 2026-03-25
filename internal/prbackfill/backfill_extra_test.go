package prbackfill

import (
	"strings"
	"testing"
)

func TestSortedStringSetKeys(t *testing.T) {
	t.Parallel()
	m := map[string]struct{}{"z": {}, "a": {}, "m": {}}
	got := sortedStringSetKeys(m)
	if len(got) != 3 || got[0] != "a" || got[1] != "m" || got[2] != "z" {
		t.Fatalf("got %v", got)
	}
}

func TestSortedStringSetKeys_empty(t *testing.T) {
	t.Parallel()
	got := sortedStringSetKeys(map[string]struct{}{})
	if len(got) != 0 {
		t.Fatalf("got %v", got)
	}
}

func TestTypeLabelNames_knownTypes(t *testing.T) {
	t.Parallel()
	present := map[string]struct{}{"feat": {}, "no-issue": {}, "fix": {}}
	got := typeLabelNames(present)
	if len(got) != 2 {
		t.Fatalf("got %v", got)
	}
	found := map[string]bool{}
	for _, n := range got {
		found[n] = true
	}
	if !found["feat"] || !found["fix"] {
		t.Fatalf("expected feat and fix, got %v", got)
	}
}

func TestTypeLabelNames_noType(t *testing.T) {
	t.Parallel()
	got := typeLabelNames(map[string]struct{}{"no-issue": {}, "hotfix": {}})
	if len(got) != 0 {
		t.Fatalf("got %v", got)
	}
}

func TestEnsureSections_emptyBody(t *testing.T) {
	t.Parallel()
	got := ensureSections("")
	if !strings.Contains(got, "## Traceability") {
		t.Fatal("missing Traceability")
	}
	if !strings.Contains(got, "## Definition of Done") {
		t.Fatal("missing DoD")
	}
	if strings.HasPrefix(got, "\n") {
		t.Fatal("should not start with newline for empty body")
	}
}

func TestEnsureSections_acceptanceCriteria(t *testing.T) {
	t.Parallel()
	body := "## Summary\n\nok\n\n## Traceability\n\nCloses #1\n\n## Risk / Impact\n\nlow\n\n## Rollback Plan\n\nrevert\n\n## Acceptance Criteria\n\n- done\n"
	got := ensureSections(body)
	if strings.Contains(got, "## Definition of Done") {
		t.Fatal("should not add DoD when Acceptance Criteria exists")
	}
}

func TestEnsureSections_closesExtracted(t *testing.T) {
	t.Parallel()
	body := "## Summary\n\nhello\n\nFixes #99\n"
	got := ensureSections(body)
	if !strings.Contains(got, "## Traceability") || !strings.Contains(got, "Fixes #99") {
		t.Fatalf("expected Traceability with Fixes, got:\n%s", got)
	}
}

func TestExtractClosesLine_noMatch(t *testing.T) {
	t.Parallel()
	if got := extractClosesLine("no link here"); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestExtractClosesLine_resolves(t *testing.T) {
	t.Parallel()
	body := "stuff\nResolves #12\nmore"
	if got := extractClosesLine(body); got != "Resolves #12" {
		t.Fatalf("got %q", got)
	}
}

func TestParseOpenPullPages_invalidJSON(t *testing.T) {
	t.Parallel()
	_, err := parseOpenPullPages([]byte("not json"))
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseOpenPullPages_emptyInput(t *testing.T) {
	t.Parallel()
	got, err := parseOpenPullPages([]byte("   "))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0, got %d", len(got))
	}
}

func TestHasHeading_variousStyles(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name  string
		body  string
		title string
		want  bool
	}{
		{"exact", "## Risk / Impact\n", "Risk / Impact", true},
		{"uppercase", "## RISK / IMPACT\n", "Risk / Impact", true},
		{"missing", "## Summary\n", "Risk / Impact", false},
		{"extra_spaces", "##  Risk / Impact  \n", "Risk / Impact", true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := hasHeading(tc.body, tc.title); got != tc.want {
				t.Fatalf("hasHeading(%q, %q) = %v, want %v", tc.body, tc.title, got, tc.want)
			}
		})
	}
}

func TestPrTypeFromTitle_allTypes(t *testing.T) {
	t.Parallel()
	for _, typ := range []string{"feat", "fix", "refactor", "perf", "test", "docs", "build", "ci", "chore", "style", "revert"} {
		t.Run(typ, func(t *testing.T) {
			t.Parallel()
			title := typ + "(scope): do something"
			if got := prTypeFromTitle(title); got != typ {
				t.Fatalf("prTypeFromTitle(%q) = %q, want %q", title, got, typ)
			}
		})
	}
}

func TestPrTypeFromTitle_breakingChange(t *testing.T) {
	t.Parallel()
	if got := prTypeFromTitle("feat!: breaking"); got != "feat" {
		t.Fatalf("got %q", got)
	}
}

func TestDesiredLabelsWithInferredType_noExistingType(t *testing.T) {
	t.Parallel()
	present := map[string]struct{}{"no-issue": {}}
	got := desiredLabelsWithInferredType(present, "docs")
	if _, ok := got["docs"]; !ok {
		t.Fatalf("expected docs label, got %v", got)
	}
	if _, ok := got["no-issue"]; !ok {
		t.Fatal("expected no-issue preserved")
	}
}

func TestMapStringSetEqual_diffSizes(t *testing.T) {
	t.Parallel()
	a := map[string]struct{}{"x": {}}
	b := map[string]struct{}{"x": {}, "y": {}}
	if mapStringSetEqual(a, b) {
		t.Fatal("expected unequal")
	}
}

func TestMapStringSetEqual_sameKeys(t *testing.T) {
	t.Parallel()
	a := map[string]struct{}{"a": {}, "b": {}}
	b := map[string]struct{}{"b": {}, "a": {}}
	if !mapStringSetEqual(a, b) {
		t.Fatal("expected equal")
	}
}
