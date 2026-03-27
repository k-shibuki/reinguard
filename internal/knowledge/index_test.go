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
	// Given: two markdown knowledge files with valid front matter
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
when:
  eval: constant
  params:
    value: true
---

# B
`))
	writeFile(t, filepath.Join(kdir, "a.md"), []byte(`---
id: first
description: A doc
triggers:
  - alpha
when:
  eval: constant
  params:
    value: true
---

# A
`))

	// When: BuildManifest runs
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
	// Then: entries sorted by path with correct schema and ids
	if m.Entries[1].ID != "second" {
		t.Fatalf("%+v", m.Entries[1])
	}
}

func TestBuildManifest_whenPropagates(t *testing.T) {
	t.Parallel()
	// Given: knowledge file with when clause containing op/path/value
	root := t.TempDir()
	kdir := filepath.Join(root, "knowledge")
	if err := os.MkdirAll(kdir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(kdir, "w.md"), []byte(`---
id: with-when
description: d
triggers:
  - t
when:
  op: eq
  path: state.kind
  value: resolved
---

# W
`))
	// When: BuildManifest runs
	m, err := BuildManifest(root, kdir)
	if err != nil {
		t.Fatal(err)
	}
	// Then: entry.When is map with expected op/path/value
	if len(m.Entries) != 1 {
		t.Fatalf("%+v", m.Entries)
	}
	if m.Entries[0].When == nil {
		t.Fatal("expected When")
	}
	got, ok := m.Entries[0].When.(map[string]any)
	if !ok {
		t.Fatalf("when type: %T", m.Entries[0].When)
	}
	if got["op"] != "eq" || got["path"] != "state.kind" || got["value"] != "resolved" {
		t.Fatalf("unexpected when: %#v", got)
	}
}

func TestBuildManifest_duplicateID(t *testing.T) {
	t.Parallel()
	// Given: two files declaring the same id
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
when:
  eval: constant
  params:
    value: true
---
`
	writeFile(t, filepath.Join(kdir, "a.md"), []byte(body))
	writeFile(t, filepath.Join(kdir, "b.md"), []byte(body))

	// When: BuildManifest runs
	_, err := BuildManifest(root, kdir)
	// Then: duplicate id error
	if err == nil || !strings.Contains(err.Error(), "duplicate id") {
		t.Fatalf("got %v", err)
	}
}

func TestBuildManifest_missingDir(t *testing.T) {
	t.Parallel()
	// Given: knowledge directory path that does not exist
	root := t.TempDir()
	// When: BuildManifest runs
	_, err := BuildManifest(root, filepath.Join(root, "nope"))
	// Then: does not exist error
	if err == nil || !strings.Contains(err.Error(), "does not exist") {
		t.Fatalf("got %v", err)
	}
}

func TestBuildManifest_emptyDir(t *testing.T) {
	t.Parallel()
	// Given: empty knowledge directory
	root := t.TempDir()
	kdir := filepath.Join(root, "knowledge")
	if err := os.MkdirAll(kdir, 0o755); err != nil {
		t.Fatal(err)
	}
	// When: BuildManifest runs
	m, err := BuildManifest(root, kdir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Then: empty manifest entries
	if len(m.Entries) != 0 {
		t.Fatalf("expected empty entries, got %d", len(m.Entries))
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
