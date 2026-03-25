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

func TestParseFrontMatter_missingOpen(t *testing.T) {
	t.Parallel()
	_, err := ParseFrontMatter([]byte("# no front matter"))
	if err == nil || !strings.Contains(err.Error(), "opening") {
		t.Fatalf("got %v", err)
	}
}

func TestParseFrontMatter_missingClose(t *testing.T) {
	t.Parallel()
	_, err := ParseFrontMatter([]byte("---\nid: x\ndescription: d\ntriggers:\n  - t\n"))
	if err == nil || !strings.Contains(err.Error(), "closing") {
		t.Fatalf("got %v", err)
	}
}

func TestParseFrontMatter_missingId(t *testing.T) {
	t.Parallel()
	md := `---
description: d
triggers:
  - t
---
`
	_, err := ParseFrontMatter([]byte(md))
	if err == nil || !strings.Contains(err.Error(), "id") {
		t.Fatalf("got %v", err)
	}
}

func TestParseFrontMatter_emptyTriggers(t *testing.T) {
	t.Parallel()
	md := `---
id: x
description: d
triggers: []
---
`
	_, err := ParseFrontMatter([]byte(md))
	if err == nil || !strings.Contains(err.Error(), "triggers") {
		t.Fatalf("got %v", err)
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
