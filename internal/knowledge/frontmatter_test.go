package knowledge

import (
	"strings"
	"testing"
)

func TestParseFrontMatter_ok(t *testing.T) {
	t.Parallel()
	// Given: markdown with valid YAML front matter
	md := `---
id: doc-a
description: Short summary
triggers:
  - one
  - two
when:
  eval: constant
  params:
    value: true
---

# Body
`
	// When: ParseFrontMatter runs
	fm, err := ParseFrontMatter([]byte(md))
	if err != nil {
		t.Fatal(err)
	}
	// Then: id, description, triggers parsed
	if fm.ID != "doc-a" || fm.Description != "Short summary" || len(fm.Triggers) != 2 {
		t.Fatalf("%+v", fm)
	}
}

func TestParseFrontMatter_errors(t *testing.T) {
	t.Parallel()
	// Given: invalid or incomplete YAML front matter fragments
	// When: ParseFrontMatter runs
	// Then: error mentions expected validation aspect (contain)
	tests := []struct {
		name    string
		input   string
		contain string
	}{
		{
			name:    "missing_open",
			input:   "# no front matter",
			contain: "opening",
		},
		{
			name:    "missing_close",
			input:   "---\nid: x\ndescription: d\ntriggers:\n  - t\n",
			contain: "closing",
		},
		{
			name: "missing_id",
			input: `---
description: d
triggers:
  - t
---
`,
			contain: "id",
		},
		{
			name: "empty_triggers",
			input: `---
id: x
description: d
triggers: []
---
`,
			contain: "triggers",
		},
		{
			name: "missing_when",
			input: `---
id: x
description: d
triggers:
  - t
---
`,
			contain: "when",
		},
		{
			name: "duplicate_trigger",
			input: `---
id: x
description: d
triggers:
  - alpha
  - Alpha
when:
  op: eq
  path: git.branch
  value: main
---
`,
			contain: "duplicate trigger",
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			_, err := ParseFrontMatter([]byte(tt.input))
			if err == nil || !strings.Contains(err.Error(), tt.contain) {
				t.Fatalf("got %v", err)
			}
		})
	}
}

func TestParseFrontMatter_whenParses(t *testing.T) {
	t.Parallel()
	// Given: markdown with valid front matter including when clause
	md := `---
id: x
description: d
triggers:
  - t
when:
  op: eq
  path: git.branch
  value: main
---
`
	// When: ParseFrontMatter runs
	fm, err := ParseFrontMatter([]byte(md))
	if err != nil {
		t.Fatal(err)
	}
	// Then: when parsed as map with expected op
	m, ok := fm.When.(map[string]any)
	if !ok || m["op"] != "eq" {
		t.Fatalf("%+v", fm.When)
	}
}

func TestParseFrontMatter_triggersSkipBlank(t *testing.T) {
	t.Parallel()
	// Given: triggers list with whitespace-only and empty entries
	md := `---
id: x
description: d
triggers:
  - "  a  "
  - ""
  - b
when:
  eval: constant
  params:
    value: true
---
`
	// When: ParseFrontMatter runs
	fm, err := ParseFrontMatter([]byte(md))
	if err != nil {
		t.Fatal(err)
	}
	// Then: blanks skipped; trimmed non-empty kept
	if len(fm.Triggers) != 2 || fm.Triggers[0] != "a" || fm.Triggers[1] != "b" {
		t.Fatalf("%v", fm.Triggers)
	}
}
