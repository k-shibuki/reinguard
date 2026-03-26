package configdir

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolve_explicit(t *testing.T) {
	t.Parallel()
	// Given: explicit relative config directory under base
	base := t.TempDir()
	want := filepath.Join(base, "cfg")
	if err := os.MkdirAll(want, 0o755); err != nil {
		t.Fatal(err)
	}
	// When: Resolve with explicit name
	got, err := Resolve(base, "cfg")
	// Then: absolute path matches
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("got %q want %q", got, want)
	}
}

func TestResolve_gitRepo(t *testing.T) {
	t.Parallel()
	// Given: git repo with .reinguard directory
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "test")
	rg := filepath.Join(dir, ".reinguard")
	if err := os.MkdirAll(rg, 0o755); err != nil {
		t.Fatal(err)
	}
	// When: Resolve with empty explicit (discover git root)
	got, err := Resolve(dir, "")
	// Then: .reinguard under root
	if err != nil {
		t.Fatal(err)
	}
	if got != rg {
		t.Fatalf("got %q want %q", got, rg)
	}
}

func TestResolve_noGitRepoIncludesHint(t *testing.T) {
	t.Parallel()
	// Given: directory that is not a git repo
	dir := t.TempDir()
	// When: Resolve runs with empty explicit name
	_, err := Resolve(dir, "")
	// Then: error hints --config-dir
	if err == nil || !strings.Contains(err.Error(), "config-dir") {
		t.Fatalf("%v", err)
	}
}

func TestResolve_emptyCwd(t *testing.T) {
	t.Parallel()
	// Given: empty cwd
	// When: Resolve runs
	_, err := Resolve("", "x")
	// Then: empty cwd error
	if err == nil || !strings.Contains(err.Error(), "empty cwd") {
		t.Fatalf("%v", err)
	}
}

func TestRepoRoot(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name   string
		cfgDir string
		want   string
	}{
		{
			name:   "dot_reinguard_layout",
			cfgDir: filepath.Join("/repo", ".reinguard"),
			want:   filepath.Clean("/repo"),
		},
		{
			name:   "flat_layout",
			cfgDir: filepath.Join("/tmp", "cfg"),
			want:   filepath.Clean(filepath.Join("/tmp", "cfg")),
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			// Given: a config directory layout
			// When: RepoRoot is called
			got := RepoRoot(tt.cfgDir)
			// Then: expected repository root is returned
			if got != tt.want {
				t.Fatalf("got %q want %q", got, tt.want)
			}
		})
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
