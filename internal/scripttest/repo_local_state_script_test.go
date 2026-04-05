package scripttest

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWithRepoLocalStateScript_SetsRepoLocalCaches(t *testing.T) {
	t.Parallel()

	script := scriptPath(t, "with-repo-local-state.sh")
	repo := setupLocalReviewRepo(t)

	out, err := runBashScript(t, repo, script, nil,
		"bash", "-lc",
		`printf '%s\n%s\n%s\n%s\n' "$PRE_COMMIT_HOME" "$XDG_CACHE_HOME" "$GOCACHE" "$GOLANGCI_LINT_CACHE"`,
	)
	if err != nil {
		t.Fatalf("with-repo-local-state: %v\n%s", err, out)
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 4 {
		t.Fatalf("expected 4 env lines, got %d: %q", len(lines), out)
	}

	want := []string{
		filepath.Join(repo, ".tmp", "pre-commit-cache"),
		filepath.Join(repo, ".tmp", "xdg-cache"),
		filepath.Join(repo, ".tmp", "go-build-cache"),
		filepath.Join(repo, ".tmp", "golangci-lint-cache"),
	}
	for i, wantPath := range want {
		if lines[i] != wantPath {
			t.Fatalf("env[%d] = %q, want %q", i, lines[i], wantPath)
		}
		if fi, statErr := os.Stat(wantPath); statErr != nil || !fi.IsDir() {
			t.Fatalf("expected directory %q, stat err=%v", wantPath, statErr)
		}
	}
}

func TestWithRepoLocalStateScript_SetsHomeSubdir(t *testing.T) {
	t.Parallel()

	script := scriptPath(t, "with-repo-local-state.sh")
	repo := setupLocalReviewRepo(t)

	out, err := runBashScript(t, repo, script, nil,
		"--home-subdir", "cr-home",
		"bash", "-lc", `printf '%s\n' "$HOME"`,
	)
	if err != nil {
		t.Fatalf("with-repo-local-state: %v\n%s", err, out)
	}

	gotHome := strings.TrimSpace(out)
	wantHome := filepath.Join(repo, ".tmp", "cr-home")
	if gotHome != wantHome {
		t.Fatalf("HOME = %q, want %q", gotHome, wantHome)
	}
	if fi, statErr := os.Stat(wantHome); statErr != nil || !fi.IsDir() {
		t.Fatalf("expected HOME directory %q, stat err=%v", wantHome, statErr)
	}
}
