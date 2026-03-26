package knowledge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/k-shibuki/reinguard/internal/config"
	"github.com/k-shibuki/reinguard/pkg/schema"
)

func TestCheckFreshness_ok(t *testing.T) {
	t.Parallel()
	// Given: manifest built from current knowledge files
	root := t.TempDir()
	kdir := filepath.Join(root, ".reinguard", "knowledge")
	if err := os.MkdirAll(kdir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(kdir, "x.md"), []byte(`---
id: only
description: d
triggers:
  - t
---
`))

	built, err := BuildManifest(root, kdir)
	if err != nil {
		t.Fatal(err)
	}
	// When: CheckFreshness compares manifest to disk
	// Then: no error
	if err := CheckFreshness(built, root, kdir); err != nil {
		t.Fatal(err)
	}
}

func TestCheckFreshness_stale(t *testing.T) {
	t.Parallel()
	// Given: on-disk knowledge and a manifest that does not match
	root := t.TempDir()
	kdir := filepath.Join(root, "knowledge")
	if err := os.MkdirAll(kdir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(kdir, "x.md"), []byte(`---
id: only
description: d
triggers:
  - t
---
`))

	stale := &config.KnowledgeManifest{
		SchemaVersion: schema.CurrentSchemaVersion,
		Entries: []config.KnowledgeManifestEntry{
			{ID: "other", Path: "knowledge/x.md", Description: "d", Triggers: []string{"t"}},
		},
	}
	// When: CheckFreshness runs
	err := CheckFreshness(stale, root, kdir)
	// Then: stale error
	if err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("got %v", err)
	}
}
