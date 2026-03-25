package observe

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestGitProvider_Collect(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init")
	runGit(t, dir, "branch", "-M", "main")

	p := NewGitProvider()
	frag, err := p.Collect(context.Background(), Options{WorkDir: dir, DefaultBranch: "main"})
	if err != nil {
		t.Fatal(err)
	}
	if frag.Signals["branch"].(string) != "main" {
		t.Fatalf("branch: %v", frag.Signals)
	}
	if frag.Signals["stash_count"].(int) != 0 {
		t.Fatalf("stash_count: %v", frag.Signals["stash_count"])
	}
	if frag.Signals["has_upstream"].(bool) {
		t.Fatal("expected no upstream")
	}
	if frag.Signals["ahead_of_upstream"].(int) != 0 || frag.Signals["behind_of_upstream"].(int) != 0 {
		t.Fatalf("ahead/behind without upstream: %v", frag.Signals)
	}
}

func TestGitProvider_stashCount(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init")
	runGit(t, dir, "branch", "-M", "main")
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, dir, "add", "f.txt")
	runGit(t, dir, "stash", "push", "-m", "wip")

	p := NewGitProvider()
	frag, err := p.Collect(context.Background(), Options{WorkDir: dir, DefaultBranch: "main"})
	if err != nil {
		t.Fatal(err)
	}
	if frag.Signals["stash_count"].(int) != 1 {
		t.Fatalf("stash_count want 1 got %v", frag.Signals["stash_count"])
	}
}

func TestGitProvider_aheadOfUpstream(t *testing.T) {
	t.Parallel()
	bare := t.TempDir()
	runGit(t, bare, "init", "--bare")

	clone := t.TempDir()
	runGit(t, clone, "clone", bare, ".")
	runGit(t, clone, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init")
	runGit(t, clone, "branch", "-M", "main")
	runGit(t, clone, "push", "-u", "origin", "main")

	if err := os.WriteFile(filepath.Join(clone, "ahead.txt"), []byte("ahead\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	runGit(t, clone, "add", "ahead.txt")
	runGit(t, clone, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "-m", "ahead")

	p := NewGitProvider()
	frag, err := p.Collect(context.Background(), Options{WorkDir: clone, DefaultBranch: "main"})
	if err != nil {
		t.Fatal(err)
	}
	if !frag.Signals["has_upstream"].(bool) {
		t.Fatal("expected upstream after push -u")
	}
	if frag.Signals["ahead_of_upstream"].(int) != 1 {
		t.Fatalf("ahead_of_upstream want 1 got %v", frag.Signals["ahead_of_upstream"])
	}
	if frag.Signals["behind_of_upstream"].(int) != 0 {
		t.Fatalf("behind_of_upstream: %v", frag.Signals["behind_of_upstream"])
	}
}

func TestGitProvider_staleRemoteBranchesCount(t *testing.T) {
	t.Parallel()
	bare := t.TempDir()
	runGit(t, bare, "init", "--bare")

	clone := t.TempDir()
	runGit(t, clone, "clone", bare, ".")
	runGit(t, clone, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init")
	runGit(t, clone, "branch", "-M", "main")
	runGit(t, clone, "push", "-u", "origin", "main")

	p := NewGitProvider()
	frag, err := p.Collect(context.Background(), Options{WorkDir: clone, DefaultBranch: "main"})
	if err != nil {
		t.Fatal(err)
	}
	n := frag.Signals["stale_remote_branches_count"].(int)
	// Only stale branches other than origin/<default> are counted.
	if n != 0 {
		t.Fatalf("stale_remote_branches_count want 0 got %d", n)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v in %s: %v\n%s", args, dir, err, out)
	}
}
