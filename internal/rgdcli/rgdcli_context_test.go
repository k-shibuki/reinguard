package rgdcli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
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
  "schema_version": "0.7.0",
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

func TestRunContextBuild_githubAuthFails_keepsStateAndRoute(t *testing.T) {
	// Given: live collect with git + github; gh auth fails; repo identity from origin only (sandbox-like)
	if runtime.GOOS == "windows" {
		t.Skip("fake gh executable is a Unix #!/bin/sh script")
	}
	tmp := t.TempDir()
	ghBin := filepath.Join(tmp, "gh")
	script := `#!/bin/sh
exit 1
`
	if err := os.WriteFile(ghBin, []byte(script), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", tmp+string(os.PathListSeparator)+os.Getenv("PATH"))

	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init")
	runGit(t, root, "branch", "-M", "main")
	runGit(t, root, "remote", "add", "origin", "git@github.com:acme/widget.git")

	cfg := filepath.Join(root, ".reinguard")
	if err := os.Mkdir(cfg, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(cfg, "reinguard.yaml"), []byte(testFixtureReinguardGitAndGitHub))
	writeFile(t, filepath.Join(cfg, "control", "states", "r.yaml"), []byte(testFixtureRulesStateIdle))
	writeFile(t, filepath.Join(cfg, "control", "routes", "r.yaml"), []byte(testFixtureControlRoutesNext))

	var buf bytes.Buffer
	app := NewApp("test")
	app.Writer = &buf
	// When: context build runs from cwd while gh auth fails but local git identity is available
	if err := app.Run([]string{"rgd", "context", "build", "--cwd", root}); err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	// Then: observation is degraded, state/route still resolve, and github.repository comes from local_git
	assertObservationDegradedWithLocalGitHubRepo(t, out)
}

func TestRunContextBuild_procedureHint_omitsWhenRouteDoesNotMatchProcFilter(t *testing.T) {
	t.Parallel()
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	writeFile(t, filepath.Join(cfgDir, "control", "states", "default.yaml"), []byte(testFixtureRulesStateIdle))
	writeFile(t, filepath.Join(cfgDir, "control", "routes", "default.yaml"), []byte(testFixtureControlRoutesNext))
	if err := os.Mkdir(filepath.Join(cfgDir, "knowledge"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(cfgDir, "knowledge", "manifest.json"), []byte(`{"schema_version":"0.7.0","entries":[]}`))
	if err := os.Mkdir(filepath.Join(cfgDir, "procedure"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(cfgDir, "procedure", "p.md"), []byte(`---
id: proc-wrong-route
purpose: Fixture procedure with route filter mismatch.
applies_to:
  state_ids:
    - Idle
  route_ids:
    - not-next
---
`))
	dir := goldenCaseDir(t, "context_build")
	obs := filepath.Join(dir, "observation.json")
	var buf bytes.Buffer
	app := NewApp("test")
	app.Writer = &buf
	if err := app.Run([]string{"rgd", "context", "build", "--config-dir", cfgDir, "--observation-file", obs}); err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	state := mustMap(t, out["state"], "state")
	if _, ok := state["procedure_hint"]; ok {
		t.Fatalf("unexpected procedure_hint: %v", state["procedure_hint"])
	}
}

func TestRunContextBuild_procedureHint_omitsWhenStateAmbiguous(t *testing.T) {
	t.Parallel()
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	writeFile(t, filepath.Join(cfgDir, "control", "states", "default.yaml"), []byte(testFixtureRulesStateAmbiguous))
	writeFile(t, filepath.Join(cfgDir, "control", "routes", "default.yaml"), []byte(testFixtureControlRoutesNext))
	if err := os.Mkdir(filepath.Join(cfgDir, "knowledge"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(cfgDir, "knowledge", "manifest.json"), []byte(`{"schema_version":"0.7.0","entries":[]}`))
	if err := os.Mkdir(filepath.Join(cfgDir, "procedure"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(cfgDir, "procedure", "p.md"), []byte(`---
id: proc-a
purpose: Maps only state A for ambiguous fixture.
applies_to:
  state_ids:
    - A
  route_ids: []
---
`))
	obsPath := filepath.Join(t.TempDir(), "obs.json")
	writeFile(t, obsPath, []byte(`{"schema_version":"0.7.0","signals":{"git":{"branch":"feat"}},"degraded":false}`))
	var buf bytes.Buffer
	app := NewApp("test")
	app.Writer = &buf
	if err := app.Run([]string{"rgd", "context", "build", "--config-dir", cfgDir, "--observation-file", obsPath}); err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	state := mustMap(t, out["state"], "state")
	if state["kind"] != "ambiguous" {
		t.Fatalf("want ambiguous state, got %v", state["kind"])
	}
	if _, ok := state["procedure_hint"]; ok {
		t.Fatal("unexpected procedure_hint for non-resolved state")
	}
}

func TestRunContextBuild_rejectsScopeFlagsWithObservationFile(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		args []string
	}{
		{name: "pr", args: []string{"--pr", "104"}},
		{name: "branch", args: []string{"--branch", "feat/test"}},
		{name: "branch and pr", args: []string{"--branch", "feat/test", "--pr", "104"}},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Given: minimal config and a pre-built observation file
			cfgDir := t.TempDir()
			writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
			writeFile(t, filepath.Join(cfgDir, "control", "states", "default.yaml"), []byte(testFixtureRulesStateIdle))
			writeFile(t, filepath.Join(cfgDir, "control", "routes", "default.yaml"), []byte(testFixtureControlRoutesNext))
			obsPath := filepath.Join(t.TempDir(), "observation.json")
			writeFile(t, obsPath, []byte(`{"schema_version":"0.7.0","signals":{"git":{"branch":"main"}},"degraded":false}`))

			// When: context build runs with --observation-file and a scope flag
			app := NewApp("test")
			args := append([]string{"rgd", "context", "build", "--config-dir", cfgDir, "--observation-file", obsPath}, tc.args...)
			err := app.Run(args)

			// Then: command rejects the scope override because the observation is already fixed
			if err == nil {
				t.Fatal("expected error when using --observation-file with scope flags, got nil")
			}
			if !strings.Contains(err.Error(), "--branch/--pr cannot be used with --observation-file") {
				t.Fatalf("expected error containing '--branch/--pr cannot be used with --observation-file', got: %v", err)
			}
		})
	}
}

func assertObservationDegradedWithLocalGitHubRepo(t *testing.T, out map[string]any) {
	t.Helper()
	obs := mustMap(t, out["observation"], "observation")
	deg, degOK := obs["degraded"].(bool)
	if !degOK || !deg {
		t.Fatalf("expected observation.degraded=true, got %v", obs["degraded"])
	}
	diags, ok := obs["diagnostics"].([]any)
	if !ok {
		t.Fatalf("expected observation.diagnostics array, got %T", obs["diagnostics"])
	}
	var sawAuthFailed bool
	for _, d := range diags {
		m, ok := d.(map[string]any)
		if !ok {
			continue
		}
		if m["code"] == "auth_failed" {
			sawAuthFailed = true
			break
		}
	}
	if !sawAuthFailed {
		t.Fatalf("expected auth_failed diagnostic in %+v", diags)
	}
	assertStateIdleRouteNext(t, out)
	signals := mustMap(t, obs["signals"], "signals")
	gh := mustMap(t, signals["github"], "signals.github")
	repo := mustMap(t, gh["repository"], "repository")
	if repo["owner"] != "acme" || repo["name"] != "widget" {
		t.Fatalf("repository: %+v", repo)
	}
	if repo["identity_source"] != "local_git" {
		t.Fatalf("identity_source: %+v", repo)
	}
}

func assertStateIdleRouteNext(t *testing.T, out map[string]any) {
	t.Helper()
	state := mustMap(t, out["state"], "state")
	if state["state_id"] != "Idle" {
		t.Fatalf("state_id: %v", state["state_id"])
	}
	routes := mustSlice(t, out["routes"], "routes")
	r0 := mustMap(t, routes[0], "routes[0]")
	if r0["route_id"] != "next" {
		t.Fatalf("route_id: %v", r0["route_id"])
	}
}

func mustMap(t *testing.T, v any, label string) map[string]any {
	t.Helper()
	m, ok := v.(map[string]any)
	if !ok {
		t.Fatalf("%s: %T", label, v)
	}
	return m
}

func mustSlice(t *testing.T, v any, label string) []any {
	t.Helper()
	s, ok := v.([]any)
	if !ok || len(s) == 0 {
		t.Fatalf("%s: %v", label, v)
	}
	return s
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v %s", args, err, string(out))
	}
}
