package rgdcli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestRunKnowledgePack_emptyManifest(t *testing.T) {
	t.Parallel()
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	if err := os.Mkdir(filepath.Join(cfgDir, "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	app := NewApp("t")
	app.Writer = &buf
	if err := app.Run([]string{"rgd", "knowledge", "pack", "--config-dir", cfgDir}); err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON output: %v; raw=%s", err, buf.String())
	}
	if _, ok := out["entries"]; !ok {
		t.Fatalf("missing 'entries' key in output: %s", buf.String())
	}
	entries, ok := out["entries"].([]any)
	if !ok {
		t.Fatalf("expected 'entries' to be array, got %T", out["entries"])
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty entries, got %v", entries)
	}
}

func TestRunKnowledgePack_queryFilter(t *testing.T) {
	t.Parallel()
	// Given: a config directory with a manifest containing two entries
	//        (entry "a" has trigger "apple"; entry "b" has trigger "banana")
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	if err := os.Mkdir(filepath.Join(cfgDir, "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(cfgDir, "knowledge"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(cfgDir, "knowledge", "manifest.json"), []byte(`{
  "schema_version": "0.3.0",
  "entries": [
    {
      "id": "a",
      "path": "knowledge/a.md",
      "description": "alpha",
      "triggers": ["apple", "alpha"]
    },
    {
      "id": "b",
      "path": "knowledge/b.md",
      "description": "beta",
      "triggers": ["banana"]
    }
  ]
}`))

	// When: running knowledge pack with --query "app"
	var buf bytes.Buffer
	app := NewApp("t")
	app.Writer = &buf
	if err := app.Run([]string{"rgd", "knowledge", "pack", "--config-dir", cfgDir, "--query", "app"}); err != nil {
		t.Fatal(err)
	}
	// Then: only entry "a" is returned (trigger "apple" contains "app")
	var out struct {
		Entries []map[string]any `json:"entries"`
	}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	if len(out.Entries) != 1 || out.Entries[0]["id"] != "a" {
		t.Fatalf("got %s", buf.String())
	}
}

func TestRunKnowledgeIndex_writesManifest(t *testing.T) {
	t.Parallel()
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	if err := os.Mkdir(filepath.Join(cfgDir, "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	kdir := filepath.Join(cfgDir, "knowledge")
	if err := os.Mkdir(kdir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(kdir, "z.md"), []byte(`---
id: zed
description: last file
triggers:
  - z
---

# Z
`))

	var buf bytes.Buffer
	app := NewApp("t")
	app.Writer = &buf
	if err := app.Run([]string{"rgd", "knowledge", "index", "--config-dir", cfgDir}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("wrote 1 entries")) {
		t.Fatalf("stdout: %s", buf.String())
	}
	data, err := os.ReadFile(filepath.Join(kdir, "manifest.json"))
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatal(err)
	}
	entries, _ := m["entries"].([]any)
	if len(entries) != 1 {
		t.Fatalf("%s", data)
	}
}

func TestRunConfigValidate_knowledgePathsAndFreshness_ok(t *testing.T) {
	t.Parallel()
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	if err := os.Mkdir(filepath.Join(cfgDir, "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	kdir := filepath.Join(cfgDir, "knowledge")
	if err := os.Mkdir(kdir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(kdir, "only.md"), []byte(`---
id: only
description: d
triggers:
  - t
---
`))
	var idxBuf bytes.Buffer
	appIdx := NewApp("t")
	appIdx.Writer = &idxBuf
	if err := appIdx.Run([]string{"rgd", "knowledge", "index", "--config-dir", cfgDir}); err != nil {
		t.Fatal(err)
	}

	var out, errBuf bytes.Buffer
	app := NewApp("t")
	app.Writer = &out
	app.ErrWriter = &errBuf
	if err := app.Run([]string{"rgd", "config", "validate", "--config-dir", cfgDir}); err != nil {
		t.Fatalf("err=%v stderr=%q", err, errBuf.String())
	}
	if !bytes.Contains(out.Bytes(), []byte("config OK")) {
		t.Fatalf("stdout=%q", out.String())
	}
}

func TestRunConfigValidate_knowledgeStaleManifest(t *testing.T) {
	t.Parallel()
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	if err := os.Mkdir(filepath.Join(cfgDir, "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	kdir := filepath.Join(cfgDir, "knowledge")
	if err := os.Mkdir(kdir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(kdir, "only.md"), []byte(`---
id: only
description: d
triggers:
  - t
---
`))
	writeFile(t, filepath.Join(kdir, "manifest.json"), []byte(`{
  "schema_version": "0.3.0",
  "entries": [{
    "id": "wrong",
    "path": "knowledge/only.md",
    "description": "d",
    "triggers": ["t"]
  }]
}`))

	var out, errBuf bytes.Buffer
	app := NewApp("t")
	app.Writer = &out
	app.ErrWriter = &errBuf
	err := app.Run([]string{"rgd", "config", "validate", "--config-dir", cfgDir})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunConfigValidate_knowledgeMissingPath(t *testing.T) {
	t.Parallel()
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	if err := os.Mkdir(filepath.Join(cfgDir, "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	kdir := filepath.Join(cfgDir, "knowledge")
	if err := os.Mkdir(kdir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(kdir, "manifest.json"), []byte(`{
  "schema_version": "0.3.0",
  "entries": [{
    "id": "ghost",
    "path": "knowledge/missing.md",
    "description": "d",
    "triggers": ["t"]
  }]
}`))

	app := NewApp("t")
	app.Writer = &bytes.Buffer{}
	app.ErrWriter = &bytes.Buffer{}
	err := app.Run([]string{"rgd", "config", "validate", "--config-dir", cfgDir})
	if err == nil {
		t.Fatal("expected error")
	}
}
