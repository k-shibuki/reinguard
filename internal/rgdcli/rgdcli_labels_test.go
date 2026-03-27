package rgdcli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunLabelsList_categories(t *testing.T) {
	t.Parallel()
	// Given: labels.yaml with type, exception, and scope labels
	// When: rgd labels list runs for each category
	// Then: JSON lists non-empty names per category
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "labels.yaml"), []byte(`schema_version: "0.1.0"
categories:
  type:
    scope: shared
    labels:
      - name: feat
        color: "A2EEEF"
        description: "t"
  exception:
    labels:
      - name: hotfix
        color: "B60205"
        description: "e"
  scope:
    labels:
      - name: epic
        color: "7057FF"
        description: "s"
`))
	var buf bytes.Buffer
	app := NewApp("t")
	app.Writer = &buf

	for _, cat := range []string{"type", "exception", "scope", "all"} {
		buf.Reset()
		if err := app.Run([]string{"rgd", "labels", "list", "--config-dir", dir, "--category", cat}); err != nil {
			t.Fatalf("category %s: %v", cat, err)
		}
		var payload struct {
			Category string   `json:"category"`
			Names    []string `json:"names"`
		}
		if err := json.Unmarshal(buf.Bytes(), &payload); err != nil {
			t.Fatalf("category %s: decode: %v body=%q", cat, err, buf.String())
		}
		if payload.Category != cat {
			t.Fatalf("category %s: got category field %q", cat, payload.Category)
		}
		if len(payload.Names) == 0 {
			t.Fatalf("category %s: empty names", cat)
		}
	}
}

func TestRunLabelsList_unknownCategory(t *testing.T) {
	t.Parallel()
	// Given: valid labels.yaml and an invalid --category value
	// When: rgd labels list runs
	// Then: error mentions unknown category
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "labels.yaml"), []byte(`schema_version: "0.1.0"
categories:
  type:
    scope: shared
    labels:
      - name: feat
        color: "A2EEEF"
        description: "t"
`))
	app := NewApp("t")
	app.Writer = &bytes.Buffer{}
	err := app.Run([]string{"rgd", "labels", "list", "--config-dir", dir, "--category", "nope"})
	if err == nil || !strings.Contains(err.Error(), "unknown --category") {
		t.Fatalf("expected unknown category error, got %v", err)
	}
}
