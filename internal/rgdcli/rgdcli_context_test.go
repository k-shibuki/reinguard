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
	root := t.TempDir()
	runGit(t, root, "init")
	runGit(t, root, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init")
	runGit(t, root, "branch", "-M", "main")
	cfg := filepath.Join(root, ".reinguard")
	if err := os.Mkdir(cfg, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(cfg, "reinguard.yaml"), []byte(testFixtureReinguardGitOnly))
	if err := os.Mkdir(filepath.Join(cfg, "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(cfg, "rules", "r.yaml"), []byte(testFixtureRulesContextBuild))

	var buf bytes.Buffer
	app := NewApp("test")
	app.Writer = &buf
	err := app.Run([]string{"rgd", "context", "build", "--cwd", root})
	if err != nil {
		t.Fatal(err)
	}
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

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v %s", args, err, string(out))
	}
}
