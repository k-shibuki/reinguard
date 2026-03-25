package prbackfill

import (
	"strings"
	"testing"
)

func TestSortedStringSetKeys(t *testing.T) {
	t.Parallel()
	// Given: a map with unordered keys
	m := map[string]struct{}{"z": {}, "a": {}, "m": {}}
	// When: sorted keys are extracted
	got := sortedStringSetKeys(m)
	// Then: keys are in alphabetical order
	if len(got) != 3 || got[0] != "a" || got[1] != "m" || got[2] != "z" {
		t.Fatalf("got %v", got)
	}
}

func TestSortedStringSetKeys_empty(t *testing.T) {
	t.Parallel()
	// Given: an empty map
	// When: sorted keys are extracted
	got := sortedStringSetKeys(map[string]struct{}{})
	// Then: result is empty
	if len(got) != 0 {
		t.Fatalf("got %v", got)
	}
}

func TestTypeLabelNames_knownTypes(t *testing.T) {
	t.Parallel()
	// Given: a label set containing type labels and a non-type label
	present := map[string]struct{}{"feat": {}, "no-issue": {}, "fix": {}}
	// When: type label names are extracted
	got := typeLabelNames(present)
	// Then: only type labels are returned
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
	// Given: a label set with no type labels
	got := typeLabelNames(map[string]struct{}{"no-issue": {}, "hotfix": {}})
	// Then: result is empty
	if len(got) != 0 {
		t.Fatalf("got %v", got)
	}
}

func TestEnsureSections_emptyBody(t *testing.T) {
	t.Parallel()
	// Given: an empty PR body
	// When: sections are ensured
	got := ensureSections("")
	// Then: required sections are injected
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
	// Given: a body that already contains "Acceptance Criteria"
	body := "## Summary\n\nok\n\n## Traceability\n\nCloses #1\n\n## Risk / Impact\n\nlow\n\n## Rollback Plan\n\nrevert\n\n## Acceptance Criteria\n\n- done\n"
	// When: sections are ensured
	got := ensureSections(body)
	// Then: DoD is not duplicated
	if strings.Contains(got, "## Definition of Done") {
		t.Fatal("should not add DoD when Acceptance Criteria exists")
	}
}

func TestEnsureSections_closesExtracted(t *testing.T) {
	t.Parallel()
	// Given: a body containing "Fixes #99" but no Traceability section
	body := "## Summary\n\nhello\n\nFixes #99\n"
	// When: sections are ensured
	got := ensureSections(body)
	// Then: Traceability section is added with the extracted close link
	if !strings.Contains(got, "## Traceability") || !strings.Contains(got, "Fixes #99") {
		t.Fatalf("expected Traceability with Fixes, got:\n%s", got)
	}
}

func TestExtractClosesLine_noMatch(t *testing.T) {
	t.Parallel()
	// Given: a body with no close/fix/resolve link
	// When: extractClosesLine is called
	// Then: empty string is returned
	if got := extractClosesLine("no link here"); got != "" {
		t.Fatalf("expected empty, got %q", got)
	}
}

func TestExtractClosesLine_resolves(t *testing.T) {
	t.Parallel()
	// Given: a body containing "Resolves #12"
	body := "stuff\nResolves #12\nmore"
	// When: extractClosesLine is called
	// Then: the matching line is returned
	if got := extractClosesLine(body); got != "Resolves #12" {
		t.Fatalf("got %q", got)
	}
}

func TestParseOpenPullPages_invalidJSON(t *testing.T) {
	t.Parallel()
	// Given: invalid JSON input
	// When: parseOpenPullPages is called
	_, err := parseOpenPullPages([]byte("not json"))
	// Then: an error is returned
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestParseOpenPullPages_emptyInput(t *testing.T) {
	t.Parallel()
	// Given: whitespace-only input
	// When: parseOpenPullPages is called
	got, err := parseOpenPullPages([]byte("   "))
	// Then: no error and empty result
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 0 {
		t.Fatalf("expected 0, got %d", len(got))
	}
}

func TestHasHeading_variousStyles(t *testing.T) {
	t.Parallel()
	// Given: various heading formats
	// When/Then: hasHeading matches case-insensitively with flexible whitespace
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
	// Given: titles with each valid Conventional Commits type
	// When/Then: the correct type is extracted
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
	// Given: a title with "!" breaking change indicator
	// Then: the base type is extracted
	if got := prTypeFromTitle("feat!: breaking"); got != "feat" {
		t.Fatalf("got %q", got)
	}
}

func TestDesiredLabelsWithInferredType_noExistingType(t *testing.T) {
	t.Parallel()
	// Given: labels without a type label, and an inferred type "docs"
	present := map[string]struct{}{"no-issue": {}}
	// When: desired labels are computed
	got := desiredLabelsWithInferredType(present, "docs")
	// Then: the inferred type is added and existing labels are preserved
	if _, ok := got["docs"]; !ok {
		t.Fatalf("expected docs label, got %v", got)
	}
	if _, ok := got["no-issue"]; !ok {
		t.Fatal("expected no-issue preserved")
	}
}

func TestMapStringSetEqual_diffSizes(t *testing.T) {
	t.Parallel()
	// Given: two maps of different sizes
	a := map[string]struct{}{"x": {}}
	b := map[string]struct{}{"x": {}, "y": {}}
	// Then: they are not equal
	if mapStringSetEqual(a, b) {
		t.Fatal("expected unequal")
	}
}

func TestMapStringSetEqual_sameKeys(t *testing.T) {
	t.Parallel()
	// Given: two maps with the same keys in different order
	a := map[string]struct{}{"a": {}, "b": {}}
	b := map[string]struct{}{"b": {}, "a": {}}
	// Then: they are equal
	if !mapStringSetEqual(a, b) {
		t.Fatal("expected equal")
	}
}
