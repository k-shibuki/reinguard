package configdir

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestResolve_explicit(t *testing.T) {
	t.Parallel()
	base := t.TempDir()
	want := filepath.Join(base, "cfg")
	if err := os.MkdirAll(want, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Resolve(base, "cfg")
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolve_gitRepo(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "test")
	rg := filepath.Join(dir, ".reinguard")
	if err := os.MkdirAll(rg, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := Resolve(dir, "")
	if err != nil {
		t.Fatal(err)
	}
	if got != rg {
		t.Fatalf("got %q want %q", got, rg)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
