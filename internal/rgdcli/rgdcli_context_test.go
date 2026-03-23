package rgdcli

import (
	"bytes"
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
	if !bytes.Contains(buf.Bytes(), []byte(`"schema_version"`)) {
		t.Fatalf("%s", buf.String())
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
