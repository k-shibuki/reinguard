package rgdcli

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
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

func runGitForObserve(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v %s", args, err, string(out))
	}
}
