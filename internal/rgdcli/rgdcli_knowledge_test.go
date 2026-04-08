package rgdcli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunKnowledgePack_emptyManifest(t *testing.T) {
	t.Parallel()
	// Given: config without knowledge manifest on disk
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	var buf bytes.Buffer
	app := NewApp("t")
	app.Writer = &buf
	// When: knowledge pack runs
	if err := app.Run([]string{"rgd", "knowledge", "pack", "--config-dir", cfgDir}); err != nil {
		t.Fatal(err)
	}
	// Then: entries key present and empty array
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

func TestRunKnowledgePack_observationFileFiltersWhen(t *testing.T) {
	t.Parallel()
	// Given: manifest with two entries having different when-clauses (on-main vs other-branch)
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	kdir := filepath.Join(cfgDir, "knowledge")
	if err := os.Mkdir(kdir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(kdir, "manifest.json"), []byte(`{
  "schema_version": "0.7.0",
  "entries": [
    {
      "id": "on-main",
      "path": "knowledge/a.md",
      "description": "d",
      "triggers": ["alpha"],
      "when": {"op": "eq", "path": "git.branch", "value": "main"}
    },
    {
      "id": "other-branch",
      "path": "knowledge/b.md",
      "description": "d",
      "triggers": ["beta"],
      "when": {"op": "eq", "path": "git.branch", "value": "feature"}
    }
  ]
}`))
	obsDir := t.TempDir()
	writeFile(t, filepath.Join(obsDir, "o.json"), []byte(`{
  "schema_version": "0.7.0",
  "signals": {"git": {"branch": "main"}},
  "degraded": false
}`))
	var buf bytes.Buffer
	app := NewApp("t")
	app.Writer = &buf
	// When: knowledge pack runs with observation where git.branch=main
	if err := app.Run([]string{
		"rgd", "knowledge", "pack", "--config-dir", cfgDir,
		"--observation-file", filepath.Join(obsDir, "o.json"),
	}); err != nil {
		t.Fatal(err)
	}
	var out struct {
		Entries []map[string]any `json:"entries"`
	}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	// Then: only "on-main" entry returned (when-clause matches)
	if len(out.Entries) != 1 || out.Entries[0]["id"] != "on-main" {
		t.Fatalf("got %s", buf.String())
	}
}

func TestRunKnowledgePack_observationFileAndQueryUnion(t *testing.T) {
	t.Parallel()
	// Given: manifest with two when-filtered entries; query matches the non-matching branch's trigger
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	kdir := filepath.Join(cfgDir, "knowledge")
	if err := os.Mkdir(kdir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(kdir, "manifest.json"), []byte(`{
  "schema_version": "0.7.0",
  "entries": [
    {
      "id": "on-main",
      "path": "knowledge/a.md",
      "description": "d",
      "triggers": ["alpha"],
      "when": {"op": "eq", "path": "git.branch", "value": "main"}
    },
    {
      "id": "other-branch",
      "path": "knowledge/b.md",
      "description": "d",
      "triggers": ["beta"],
      "when": {"op": "eq", "path": "git.branch", "value": "feature"}
    }
  ]
}`))
	obsDir := t.TempDir()
	writeFile(t, filepath.Join(obsDir, "o.json"), []byte(`{
  "schema_version": "0.7.0",
  "signals": {"git": {"branch": "main"}},
  "degraded": false
}`))
	var buf bytes.Buffer
	app := NewApp("t")
	app.Writer = &buf
	// When: pack runs with same observation plus --query matching "beta" trigger
	if err := app.Run([]string{
		"rgd", "knowledge", "pack", "--config-dir", cfgDir,
		"--observation-file", filepath.Join(obsDir, "o.json"),
		"--query", "bet",
	}); err != nil {
		t.Fatal(err)
	}
	var out struct {
		Entries []map[string]any `json:"entries"`
	}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	// Then: both entries returned (OR union of when match and trigger match)
	if len(out.Entries) != 2 {
		t.Fatalf("want OR union, got %s", buf.String())
	}
}

func TestRunKnowledgePack_rejectsInvalidScopePR(t *testing.T) {
	t.Parallel()
	// Given: knowledge pack with --pr 0 (invalid) — scope validation must run without --observation-file.
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	if err := os.Mkdir(filepath.Join(cfgDir, "knowledge"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(cfgDir, "knowledge", "manifest.json"), []byte(`{
  "schema_version": "0.7.0",
  "entries": []
}`))
	app := NewApp("t")
	app.Writer = &bytes.Buffer{}
	err := app.Run([]string{"rgd", "knowledge", "pack", "--config-dir", cfgDir, "--pr", "0"})
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "--pr must be greater than 0") {
		t.Fatalf("got %v", err)
	}
}

func TestRunKnowledgePack_queryFilter(t *testing.T) {
	t.Parallel()
	// Given: a config directory with a manifest containing two entries
	//        (entry "a" has trigger "apple"; entry "b" has trigger "banana")
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	if err := os.Mkdir(filepath.Join(cfgDir, "knowledge"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(cfgDir, "knowledge", "manifest.json"), []byte(`{
  "schema_version": "0.7.0",
  "entries": [
    {
      "id": "a",
      "path": "knowledge/a.md",
      "description": "alpha",
      "triggers": ["apple", "alpha"],
      "when": {"eval": "constant", "params": {"value": true}}
    },
    {
      "id": "b",
      "path": "knowledge/b.md",
      "description": "beta",
      "triggers": ["banana"],
      "when": {"eval": "constant", "params": {"value": true}}
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
	// Given: one markdown knowledge file under knowledge/
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	kdir := filepath.Join(cfgDir, "knowledge")
	if err := os.Mkdir(kdir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(kdir, "z.md"), []byte(`---
id: zed
description: last file
triggers:
  - z
when:
  eval: constant
  params:
    value: true
---

# Z
`))

	var buf bytes.Buffer
	app := NewApp("t")
	app.Writer = &buf
	// When: knowledge index runs
	if err := app.Run([]string{"rgd", "knowledge", "index", "--config-dir", cfgDir}); err != nil {
		t.Fatal(err)
	}
	// Then: stdout reports write; manifest.json lists one entry
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
	// Given: knowledge file indexed to a fresh manifest
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	kdir := filepath.Join(cfgDir, "knowledge")
	if err := os.Mkdir(kdir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(kdir, "only.md"), []byte(`---
id: only
description: d
triggers:
  - t
when:
  eval: constant
  params:
    value: true
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
	// When: config validate runs
	if err := app.Run([]string{"rgd", "config", "validate", "--config-dir", cfgDir}); err != nil {
		t.Fatalf("err=%v stderr=%q", err, errBuf.String())
	}
	// Then: success on stdout
	if !bytes.Contains(out.Bytes(), []byte("config OK")) {
		t.Fatalf("stdout=%q", out.String())
	}
}

func TestRunConfigValidate_knowledgeStaleManifest(t *testing.T) {
	t.Parallel()
	// Given: on-disk knowledge and manifest listing wrong entry id
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	kdir := filepath.Join(cfgDir, "knowledge")
	if err := os.Mkdir(kdir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(kdir, "only.md"), []byte(`---
id: only
description: d
triggers:
  - t
when:
  eval: constant
  params:
    value: true
---
`))
	writeFile(t, filepath.Join(kdir, "manifest.json"), []byte(`{
  "schema_version": "0.7.0",
  "entries": [{
    "id": "wrong",
    "path": "knowledge/only.md",
    "description": "d",
    "triggers": ["t"],
    "when": {"eval": "constant", "params": {"value": true}}
  }]
}`))

	var out, errBuf bytes.Buffer
	app := NewApp("t")
	app.Writer = &out
	app.ErrWriter = &errBuf
	// When: config validate runs
	err := app.Run([]string{"rgd", "config", "validate", "--config-dir", cfgDir})
	// Then: error (stale manifest)
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunConfigValidate_knowledgeMissingPath(t *testing.T) {
	t.Parallel()
	// Given: manifest references a missing markdown path
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	kdir := filepath.Join(cfgDir, "knowledge")
	if err := os.Mkdir(kdir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(kdir, "manifest.json"), []byte(`{
  "schema_version": "0.7.0",
  "entries": [{
    "id": "ghost",
    "path": "knowledge/missing.md",
    "description": "d",
    "triggers": ["t"],
    "when": {"eval": "constant", "params": {"value": true}}
  }]
}`))

	app := NewApp("t")
	app.Writer = &bytes.Buffer{}
	app.ErrWriter = &bytes.Buffer{}
	// When: config validate runs
	err := app.Run([]string{"rgd", "config", "validate", "--config-dir", cfgDir})
	// Then: error
	if err == nil {
		t.Fatal("expected error")
	}
}
