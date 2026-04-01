package rgdcli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRunContextBuild_gitOnly(t *testing.T) {
	t.Parallel()
	// Given: git repo on main with minimal .reinguard, state+route rules
	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init")
	runGit(t, root, "branch", "-M", "main")
	cfg := filepath.Join(root, ".reinguard")
	if err := os.Mkdir(cfg, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(cfg, "reinguard.yaml"), []byte(testFixtureReinguardGitOnly))
	writeFile(t, filepath.Join(cfg, "control", "states", "r.yaml"), []byte(testFixtureRulesStateIdle))
	writeFile(t, filepath.Join(cfg, "control", "routes", "r.yaml"), []byte(testFixtureControlRoutesNext))

	var buf bytes.Buffer
	app := NewApp("test")
	app.Writer = &buf
	// When: context build runs from cwd
	err := app.Run([]string{"rgd", "context", "build", "--cwd", root})
	if err != nil {
		t.Fatal(err)
	}
	// Then: JSON has schema_version and empty knowledge.entries
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON output: %v; raw=%s", err, buf.String())
	}
	if _, ok := out["schema_version"]; !ok {
		t.Fatalf("missing schema_version: %v", out)
	}
	knowledge, ok := out["knowledge"].(map[string]any)
	if !ok {
		t.Fatalf("knowledge is not object: %T", out["knowledge"])
	}
	entries, ok := knowledge["entries"].([]any)
	if !ok {
		t.Fatalf("knowledge.entries must be array, got %T", knowledge["entries"])
	}
	if len(entries) != 0 {
		t.Fatalf("expected empty knowledge.entries, got %v", entries)
	}
}

func TestRunContextBuild_knowledgeWhenExcludes(t *testing.T) {
	t.Parallel()
	// Given: config with a knowledge entry whose when-clause requires git.branch=not-main
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	writeFile(t, filepath.Join(cfgDir, "control", "states", "default.yaml"), []byte(testFixtureRulesStateIdle))
	writeFile(t, filepath.Join(cfgDir, "control", "routes", "default.yaml"), []byte(testFixtureControlRoutesNext))
	kdir := filepath.Join(cfgDir, "knowledge")
	if err := os.Mkdir(kdir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(kdir, "doc.md"), []byte(`---
id: doc1
description: d
triggers:
  - fixture
when:
  op: eq
  path: git.branch
  value: not-main
---
`))
	writeFile(t, filepath.Join(kdir, "manifest.json"), []byte(`{
  "schema_version": "0.6.1",
  "entries": [{
    "id": "doc1",
    "path": "knowledge/doc.md",
    "description": "d",
    "triggers": ["fixture"],
    "when": {"op": "eq", "path": "git.branch", "value": "not-main"}
  }]
}`))
	dir := goldenCaseDir(t, "context_build")
	obs := filepath.Join(dir, "observation.json")

	var buf bytes.Buffer
	app := NewApp("test")
	app.Writer = &buf
	// When: context build runs with observation where git.branch=main
	if err := app.Run([]string{"rgd", "context", "build", "--config-dir", cfgDir, "--observation-file", obs}); err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	km, ok := out["knowledge"].(map[string]any)
	if !ok {
		t.Fatalf("knowledge: %T", out["knowledge"])
	}
	entries, ok := km["entries"].([]any)
	if !ok {
		t.Fatalf("entries: %T", km["entries"])
	}
	// Then: knowledge.entries is empty (when-clause does not match)
	if len(entries) != 0 {
		t.Fatalf("expected no knowledge entries, got %v", entries)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v %s", args, err, string(out))
	}
}
