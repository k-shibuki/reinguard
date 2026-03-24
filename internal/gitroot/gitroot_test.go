package gitroot

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestRoot_gitInitRepo(t *testing.T) {
	t.Parallel()
	// Given: git repo with nested cwd
	dir := t.TempDir()
	run(t, dir, "git", "init")
	run(t, dir, "git", "config", "user.email", "test@example.com")
	run(t, dir, "git", "config", "user.name", "test")
	sub := filepath.Join(dir, "nested")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	// When: Root from nested path
	root, err := Root(sub)
	// Then: toplevel matches repo root
	if err != nil {
		t.Fatalf("Root: %v", err)
	}
	wantRoot, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	gotRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatalf("EvalSymlinks: %v", err)
	}
	if gotRoot != wantRoot {
		t.Fatalf("got root %q want %q", gotRoot, wantRoot)
	}
}

func TestRoot_notGitRepo(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	_, err := Root(dir)
	if err == nil {
		t.Fatal("expected error for non-git directory")
	}
	if !strings.Contains(err.Error(), "not a git repository") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRoot_emptyCwd(t *testing.T) {
	t.Parallel()
	_, err := Root("")
	if err == nil {
		t.Fatal("expected error for empty cwd")
	}
	if !strings.Contains(err.Error(), "empty cwd") {
		t.Fatalf("unexpected error: %v", err)
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
