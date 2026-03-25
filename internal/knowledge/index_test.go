package knowledge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/k-shibuki/reinguard/pkg/schema"
)

func TestBuildManifest_ok(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	kdir := filepath.Join(root, ".reinguard", "knowledge")
	if err := os.MkdirAll(kdir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(kdir, "b.md"), []byte(`---
id: second
description: B doc
triggers:
  - beta
---

# B
`))
	writeFile(t, filepath.Join(kdir, "a.md"), []byte(`---
id: first
description: A doc
triggers:
  - alpha
---

# A
`))

	m, err := BuildManifest(root, kdir)
	if err != nil {
		t.Fatal(err)
	}
	if m.SchemaVersion != schema.CurrentSchemaVersion {
		t.Fatalf("schema %q", m.SchemaVersion)
	}
	if len(m.Entries) != 2 {
		t.Fatalf("entries %+v", m.Entries)
	}
	// Sorted by filename: a.md then b.md
	if m.Entries[0].ID != "first" || m.Entries[0].Path != ".reinguard/knowledge/a.md" {
		t.Fatalf("%+v", m.Entries[0])
	}
	if m.Entries[1].ID != "second" {
		t.Fatalf("%+v", m.Entries[1])
	}
}

func TestBuildManifest_duplicateID(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	kdir := filepath.Join(root, "knowledge")
	if err := os.MkdirAll(kdir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `---
id: same
description: x
triggers:
  - t
---
`
	writeFile(t, filepath.Join(kdir, "a.md"), []byte(body))
	writeFile(t, filepath.Join(kdir, "b.md"), []byte(body))

	_, err := BuildManifest(root, kdir)
	if err == nil || !strings.Contains(err.Error(), "duplicate id") {
		t.Fatalf("got %v", err)
	}
}

func TestBuildManifest_missingDir(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	_, err := BuildManifest(root, filepath.Join(root, "nope"))
	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("got %v", err)
	}
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
