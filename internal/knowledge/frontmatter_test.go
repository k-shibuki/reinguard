package knowledge

import (
	"strings"
	"testing"
)

func TestParseFrontMatter_ok(t *testing.T) {
	t.Parallel()
	md := `---
id: doc-a
description: Short summary
triggers:
  - one
  - two
---

# Body
`
	fm, err := ParseFrontMatter([]byte(md))
	if err != nil {
		t.Fatal(err)
	}
	if fm.ID != "doc-a" || fm.Description != "Short summary" || len(fm.Triggers) != 2 {
		t.Fatalf("%+v", fm)
	}
}

func TestParseFrontMatter_errors(t *testing.T) {
	t.Parallel()
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
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Given: invalid or incomplete front matter
			// When: ParseFrontMatter is called
			_, err := ParseFrontMatter([]byte(tt.input))
			// Then: error mentions the expected validation aspect
			if err == nil || !strings.Contains(err.Error(), tt.contain) {
				t.Fatalf("got %v", err)
			}
		})
	}
}

func TestParseFrontMatter_triggersSkipBlank(t *testing.T) {
	t.Parallel()
	md := `---
id: x
description: d
triggers:
  - "  a  "
  - ""
  - b
---
`
	fm, err := ParseFrontMatter([]byte(md))
	if err != nil {
		t.Fatal(err)
	}
	if len(fm.Triggers) != 2 || fm.Triggers[0] != "a" || fm.Triggers[1] != "b" {
		t.Fatalf("%v", fm.Triggers)
	}
}
