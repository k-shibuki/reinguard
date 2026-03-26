package prbackfill

import (
	"strings"
	"testing"
)

func TestParseOpenPullPages_singleArray(t *testing.T) {
	// Given: JSON array of one PR object
	raw := `[{"number":1,"title":"feat: x","body":null,"labels":[]}]`
	// When: parseOpenPullPages runs
	got, err := parseOpenPullPages([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	// Then: one PR with expected fields
	if len(got) != 1 || got[0].Number != 1 || got[0].Title != "feat: x" {
		t.Fatalf("got %+v", got)
	}
}

func TestParseOpenPullPages_twoArrays(t *testing.T) {
	// Given: two JSON arrays concatenated in one payload
	raw := `[{"number":1,"title":"a","body":null,"labels":[]}]  [{"number":2,"title":"b","body":null,"labels":[]}]`
	// When: parseOpenPullPages runs
	got, err := parseOpenPullPages([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	// Then: both PRs merged in order
	if len(got) != 2 || got[0].Number != 1 || got[1].Number != 2 {
		t.Fatalf("got %+v", got)
	}
}

func TestHasHeading_caseInsensitive(t *testing.T) {
	// Given: body with mixed-case heading
	body := "## TRACEABILITY\n\nCloses #1\n"
	// When/Then: hasHeading matches canonical title case-insensitively
	if !hasHeading(body, "Traceability") {
		t.Fatal("expected match")
	}
}

func TestPrTypeFromTitle(t *testing.T) {
	// Given/When/Then: table of titles and expected Conventional Commit types
	tests := []struct {
		title, want string
	}{
		{"feat: add x", "feat"},
		{"fix(scope): y", "fix"},
		{"hotfix: z", ""},
		{"no conventional", ""},
	}
	for _, tc := range tests {
		if got := prTypeFromTitle(tc.title); got != tc.want {
			t.Errorf("%q: got %q want %q", tc.title, got, tc.want)
		}
	}
}

func TestExtractClosesLine(t *testing.T) {
	// Given: body containing a Closes line
	body := "x\n\nCloses #42\n"
	// When: extractClosesLine runs
	// Then: full line returned
	if got := extractClosesLine(body); got != "Closes #42" {
		t.Fatalf("got %q", got)
	}
}

func TestEnsureSections_idempotentWhenPresent(t *testing.T) {
	// Given: body already has all required sections
	body := `## Summary

x

## Traceability

Closes #1

## Risk / Impact

- ok

## Rollback Plan

N/A

## Definition of Done

- [ ] done
`
	// When: ensureSections runs
	got := ensureSections(body)
	// Then: unchanged aside from whitespace normalization expectation
	if strings.TrimSpace(got) != strings.TrimSpace(body) {
		t.Fatalf("expected unchanged, got:\n%s", got)
	}
}

func TestEnsureSections_addsMissing(t *testing.T) {
	// Given: body with only Summary
	// When: ensureSections runs
	got := ensureSections("## Summary\n\nhello\n")
	// Then: standard sections are injected
	if !strings.Contains(got, "## Traceability") {
		t.Fatal("missing Traceability")
	}
	if !strings.Contains(got, "## Risk / Impact") {
		t.Fatal("missing Risk")
	}
	if !strings.Contains(got, "## Rollback Plan") {
		t.Fatal("missing Rollback")
	}
	if !strings.Contains(got, "## Definition of Done") {
		t.Fatal("missing DoD")
	}
}

func TestDesiredLabelsWithInferredType(t *testing.T) {
	// Given: existing labels including a type label and inferred type "fix"
	present := map[string]struct{}{
		"feat": {}, "no-issue": {}, "bug": {},
	}
	// When: desiredLabelsWithInferredType runs
	got := desiredLabelsWithInferredType(present, "fix")
	// Then: old type replaced, non-type labels kept
	want := map[string]struct{}{"no-issue": {}, "bug": {}, "fix": {}}
	if !mapStringSetEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestMapStringSetEqual(t *testing.T) {
	// Given: two maps with same keys
	a := map[string]struct{}{"x": {}, "y": {}}
	b := map[string]struct{}{"y": {}, "x": {}}
	// When/Then: equal regardless of map iteration order
	if !mapStringSetEqual(a, b) {
		t.Fatal("expected equal")
	}
	if mapStringSetEqual(a, map[string]struct{}{"x": {}}) {
		t.Fatal("expected unequal")
	}
}
