package procedure

import (
	"strings"
	"testing"
)

func TestParseFrontMatter_ok(t *testing.T) {
	t.Parallel()
	md := `---
id: procedure-test
purpose: Test purpose.
applies_to:
  state_ids:
    - working_no_pr
  route_ids:
    - user-implement
reads:
  - ../policy/coding--standards.md
  - ../policy/commit--format.md
---
# Body
`
	fm, err := ParseFrontMatter([]byte(md))
	if err != nil {
		t.Fatal(err)
	}
	if fm.ID != "procedure-test" || fm.Purpose != "Test purpose." {
		t.Fatalf("%+v", fm)
	}
	if len(fm.AppliesTo.StateIDs) != 1 || fm.AppliesTo.StateIDs[0] != "working_no_pr" {
		t.Fatalf("state_ids %+v", fm.AppliesTo.StateIDs)
	}
	if len(fm.AppliesTo.RouteIDs) != 1 || fm.AppliesTo.RouteIDs[0] != "user-implement" {
		t.Fatalf("route_ids %+v", fm.AppliesTo.RouteIDs)
	}
	if len(fm.Reads) != 2 ||
		fm.Reads[0] != "../policy/coding--standards.md" ||
		fm.Reads[1] != "../policy/commit--format.md" {
		t.Fatalf("reads %+v", fm.Reads)
	}
}

func TestParseFrontMatter_emptyAppliesTo(t *testing.T) {
	t.Parallel()
	md := `---
id: procedure-meta
purpose: Orchestration only.
applies_to:
  state_ids: []
  route_ids: []
---
`
	fm, err := ParseFrontMatter([]byte(md))
	if err != nil {
		t.Fatal(err)
	}
	if len(fm.AppliesTo.StateIDs) != 0 || len(fm.AppliesTo.RouteIDs) != 0 {
		t.Fatalf("%+v", fm.AppliesTo)
	}
}

func TestParseFrontMatter_emptyReads(t *testing.T) {
	t.Parallel()
	md := `---
id: procedure-empty-reads
purpose: Empty reads.
applies_to:
  state_ids: []
  route_ids: []
reads: []
---
`
	fm, err := ParseFrontMatter([]byte(md))
	if err != nil {
		t.Fatal(err)
	}
	if len(fm.Reads) != 0 {
		t.Fatalf("reads %+v", fm.Reads)
	}
}

func TestParseFrontMatter_singleRead(t *testing.T) {
	t.Parallel()
	md := `---
id: procedure-single-read
purpose: Single read.
applies_to:
  state_ids: []
  route_ids: []
reads:
  - ../policy/one.md
---
`
	fm, err := ParseFrontMatter([]byte(md))
	if err != nil {
		t.Fatal(err)
	}
	if len(fm.Reads) != 1 || fm.Reads[0] != "../policy/one.md" {
		t.Fatalf("reads %+v", fm.Reads)
	}
}

func TestParseFrontMatter_blockScalarAllowsIndentedDelimiters(t *testing.T) {
	t.Parallel()
	md := `---
id: procedure-block-scalar
purpose: |
  Keep the literal line below as part of the YAML value.
  ---
  More text.
applies_to:
  state_ids: []
  route_ids: []
---
`
	fm, err := ParseFrontMatter([]byte(md))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(fm.Purpose, "More text.") {
		t.Fatalf("purpose=%q", fm.Purpose)
	}
}

func TestParseFrontMatter_errors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		input   string
		contain string
	}{
		{name: "missing_open", input: "# no fm", contain: "opening"},
		{name: "missing_close", input: "---\nid: x\npurpose: p\n", contain: "closing"},
		{name: "missing_id", input: "---\npurpose: p\napplies_to:\n  state_ids: []\n  route_ids: []\n---\n", contain: "id"},
		{name: "missing_purpose", input: "---\nid: x\napplies_to:\n  state_ids: []\n  route_ids: []\n---\n", contain: "purpose"},
		{name: "empty_reads_entry", input: "---\nid: x\npurpose: p\napplies_to:\n  state_ids: []\n  route_ids: []\nreads:\n  - '  '\n---\n", contain: "reads[0]"},
		{name: "empty_reads_entry_as_empty_string", input: "---\nid: x\npurpose: p\napplies_to:\n  state_ids: []\n  route_ids: []\nreads:\n  - ''\n  - ''\n---\n", contain: "reads[0]"},
		{name: "dup_reads_entry", input: "---\nid: x\npurpose: p\napplies_to:\n  state_ids: []\n  route_ids: []\nreads:\n  - ../a.md\n  - ../a.md\n---\n", contain: "duplicate read"},
		{name: "dup_state_in_file", input: `---
id: x
purpose: p
applies_to:
  state_ids:
    - a
    - a
  route_ids: []
---
`, contain: "duplicate state_id"},
		{name: "dup_route_in_file", input: `---
id: x
purpose: p
applies_to:
  state_ids: []
  route_ids:
    - r
    - r
---
`, contain: "duplicate route_id"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseFrontMatter([]byte(tt.input))
			if err == nil || !strings.Contains(err.Error(), tt.contain) {
				t.Fatalf("got %v", err)
			}
		})
	}
}
