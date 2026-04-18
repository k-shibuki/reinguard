package rgdcli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunObserve_gitOnlyProvider(t *testing.T) {
	t.Parallel()
	// Given: git repo with .reinguard and git-only provider enabled
	root := t.TempDir()
	runGitForObserve(t, root, "init")
	runGitForObserve(t, root, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init")
	cfg := filepath.Join(root, ".reinguard")
	if err := os.MkdirAll(cfg, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(cfg, "reinguard.yaml"), []byte(testFixtureReinguardGitOnly))
	writeFile(t, filepath.Join(cfg, "control", "guards", "d.yaml"), []byte(testFixtureRulesEmpty))

	var buf bytes.Buffer
	app := NewApp("t")
	app.Writer = &buf
	// When: observe git runs from repo root
	if err := app.Run([]string{"rgd", "observe", "git", "--cwd", root}); err != nil {
		t.Fatal(err)
	}
	// Then: stdout contains git provider and signals
	if !bytes.Contains(buf.Bytes(), []byte(`"git"`)) || !bytes.Contains(buf.Bytes(), []byte(`"signals"`)) {
		t.Fatalf("%s", buf.String())
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	meta, ok := out["meta"].(map[string]any)
	if !ok {
		t.Fatalf("want meta to be map[string]any, got %T: %v", out["meta"], out["meta"])
	}
	if meta["view"] != "summary" {
		t.Fatalf("want meta.view=summary, got %v", meta["view"])
	}
}

func TestRunObserve_providerOverridePreservesConfiguredOptions(t *testing.T) {
	t.Parallel()
	// Given: a repo config where the github provider has invalid options.bot_reviewers shape
	root := t.TempDir()
	runGitForObserve(t, root, "init")
	runGitForObserve(t, root, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init")
	cfg := filepath.Join(root, ".reinguard")
	if err := os.MkdirAll(cfg, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(cfg, "reinguard.yaml"), []byte(`schema_version: "0.7.0"
default_branch: main
providers:
  - id: git
    enabled: true
  - id: github
    enabled: true
    options:
      bot_reviewers:
        - id: coderabbit
          login: "coderabbitai[bot]"
          required: true
          enrich: ["unknown-enrich"]
`))
	writeFile(t, filepath.Join(cfg, "control", "guards", "d.yaml"), []byte(testFixtureRulesEmpty))

	var buf bytes.Buffer
	app := NewApp("t")
	app.Writer = &buf
	// When: observe github selects the configured github provider via provider override
	err := app.Run([]string{"rgd", "observe", "github", "--cwd", root})
	// Then: provider options are still validated (not discarded by the override path)
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !strings.Contains(err.Error(), "unknown enrich") || !strings.Contains(err.Error(), "unknown-enrich") {
		t.Fatalf("expected unknown enrich validation error, got: %v", err)
	}
}

func runGitForObserve(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v %s", args, err, string(out))
	}
}
