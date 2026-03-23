package gitroot

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRoot_gitInitRepo(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	run(t, dir, "git", "init")
	run(t, dir, "git", "config", "user.email", "test@example.com")
	run(t, dir, "git", "config", "user.name", "test")
	sub := filepath.Join(dir, "nested")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	root, err := Root(sub)
	if err != nil {
		t.Fatalf("Root: %v", err)
	}
	if root != dir {
		t.Fatalf("got root %q want %q", root, dir)
	}
}

func TestRoot_notGitRepo(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	_, err := Root(dir)
	if err == nil {
		t.Fatal("expected error outside git repo")
	}
}

func run(t *testing.T, dir string, name string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v\n%s", name, args, err, out)
	}
}
