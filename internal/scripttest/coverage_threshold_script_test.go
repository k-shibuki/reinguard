package scripttest

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckCoverageThresholdScript(t *testing.T) {
	root := repoRoot(t)
	script := scriptPath(t, "check-coverage-threshold.sh")

	// Given/When/Then: coverage profiles are checked against the requested minimum threshold.
	t.Run("missingProfileFails", func(t *testing.T) {
		out, err := runBashScript(t, root, script, nil, "80", filepath.Join(t.TempDir(), "missing.out"))
		if err == nil {
			t.Fatalf("expected error, got success:\n%s", out)
		}
		if !strings.Contains(out, "coverage profile not found") {
			t.Fatalf("expected missing profile error, got:\n%s", out)
		}
	})

	t.Run("thresholdEvaluation", func(t *testing.T) {
		dir := t.TempDir()
		goFile := writeTempFile(t, dir, "sample-*.go", "package sample\n\nfunc Covered() {}\nfunc Uncovered() {}\n")
		profile := writeTempFile(t, dir, "coverage-*.out", "mode: set\n"+goFile+":3.1,3.17 1 1\n"+goFile+":4.1,4.19 1 0\n")

		out, err := runBashScript(t, root, script, nil, "40", profile)
		if err != nil {
			t.Fatalf("expected success at 40%% threshold: %v\n%s", err, out)
		}
		if !strings.Contains(out, "total coverage:") {
			t.Fatalf("expected coverage summary, got:\n%s", out)
		}

		out, err = runBashScript(t, root, script, nil, "60", profile)
		if err == nil {
			t.Fatalf("expected threshold failure, got success:\n%s", out)
		}
		if !strings.Contains(out, "below required 60.0%") {
			t.Fatalf("expected threshold failure message, got:\n%s", out)
		}
	})
}
