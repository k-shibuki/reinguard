package rgdcli

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRunObserve_gitOnlyProvider(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	runGitForObserve(t, root, "init")
	runGitForObserve(t, root, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init")
	cfg := filepath.Join(root, ".reinguard")
	if err := os.MkdirAll(cfg, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(cfg, "reinguard.yaml"), []byte(testFixtureReinguardGitOnly))
	if err := os.MkdirAll(filepath.Join(cfg, "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(cfg, "rules", "d.yaml"), []byte(testFixtureRulesEmpty))

	var buf bytes.Buffer
	app := NewApp("t")
	app.Writer = &buf
	if err := app.Run([]string{"rgd", "observe", "git", "--cwd", root}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"git"`)) || !bytes.Contains(buf.Bytes(), []byte(`"signals"`)) {
		t.Fatalf("%s", buf.String())
	}
}

func runGitForObserve(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v %s", args, err, string(out))
	}
}
