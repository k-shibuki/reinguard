package schema

import "testing"

func TestListEmbedded_nonEmpty(t *testing.T) {
	t.Parallel()
	// Given: embedded schema FS is present
	// When: ListEmbedded walks embed FS
	names, err := ListEmbedded()
	// Then: multiple schema files are listed
	if err != nil {
		t.Fatal(err)
	}
	required := map[string]struct{}{
		"reinguard-config.json":     {},
		"rules-document.json":       {},
		"observation-document.json": {},
		"operational-context.json":  {},
		"knowledge-manifest.json":   {},
		"labels-config.json":        {},
	}
	got := make(map[string]struct{}, len(names))
	for _, n := range names {
		got[n] = struct{}{}
	}
	for n := range required {
		if _, ok := got[n]; !ok {
			t.Fatalf("missing embedded schema: %s (got=%v)", n, names)
		}
	}
}
