package prbackfill

import (
	"strings"
	"testing"
)

func TestParseOpenPullPages_singleArray(t *testing.T) {
	raw := `[{"number":1,"title":"feat: x","body":null,"labels":[]}]`
	got, err := parseOpenPullPages([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Number != 1 || got[0].Title != "feat: x" {
		t.Fatalf("got %+v", got)
	}
}

func TestParseOpenPullPages_twoArrays(t *testing.T) {
	raw := `[{"number":1,"title":"a","body":null,"labels":[]}]  [{"number":2,"title":"b","body":null,"labels":[]}]`
	got, err := parseOpenPullPages([]byte(raw))
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Number != 1 || got[1].Number != 2 {
		t.Fatalf("got %+v", got)
	}
}

func TestHasHeading_caseInsensitive(t *testing.T) {
	body := "## TRACEABILITY\n\nCloses #1\n"
	if !hasHeading(body, "Traceability") {
		t.Fatal("expected match")
	}
}

func TestPrTypeFromTitle(t *testing.T) {
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
	body := "x\n\nCloses #42\n"
	if got := extractClosesLine(body); got != "Closes #42" {
		t.Fatalf("got %q", got)
	}
}

func TestEnsureSections_idempotentWhenPresent(t *testing.T) {
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
	got := ensureSections(body)
	if strings.TrimSpace(got) != strings.TrimSpace(body) {
		t.Fatalf("expected unchanged, got:\n%s", got)
	}
}

func TestEnsureSections_addsMissing(t *testing.T) {
	got := ensureSections("## Summary\n\nhello\n")
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
	present := map[string]struct{}{
		"feat": {}, "no-issue": {}, "bug": {},
	}
	got := desiredLabelsWithInferredType(present, "fix")
	want := map[string]struct{}{"no-issue": {}, "bug": {}, "fix": {}}
	if !mapStringSetEqual(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}

func TestMapStringSetEqual(t *testing.T) {
	a := map[string]struct{}{"x": {}, "y": {}}
	b := map[string]struct{}{"y": {}, "x": {}}
	if !mapStringSetEqual(a, b) {
		t.Fatal("expected equal")
	}
	if mapStringSetEqual(a, map[string]struct{}{"x": {}}) {
		t.Fatal("expected unequal")
	}
}
