package gitroot

import (
	"context"
	"os/exec"
	"strings"
	"testing"
)

func TestCurrentBranch_named(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	runGitBranchTest(t, dir, "init")
	runGitBranchTest(t, dir, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init")
	runGitBranchTest(t, dir, "branch", "-M", "main")

	b, detached, err := CurrentBranch(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if detached || b != "main" {
		t.Fatalf("got branch=%q detached=%v", b, detached)
	}
}

func TestCurrentBranch_nonGitDir(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	_, _, err := CurrentBranch(context.Background(), dir)
	if err == nil {
		t.Fatal("expected error for non-git directory")
	}
}

func TestCurrentBranch_detached(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	runGitBranchTest(t, dir, "init")
	runGitBranchTest(t, dir, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init")
	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	commit := strings.TrimSpace(string(out))
	runGitBranchTest(t, dir, "checkout", "--detach", commit)

	b, detached, err := CurrentBranch(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if !detached || b != "" {
		t.Fatalf("got branch=%q detached=%v", b, detached)
	}
}

func runGitBranchTest(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
