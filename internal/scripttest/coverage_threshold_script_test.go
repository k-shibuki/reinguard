package scripttest

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestCheckCoverageThresholdScript(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	script := scriptPath(t, "check-coverage-threshold.sh")

	// Given/When/Then: coverage profiles are checked against the requested minimum threshold.
	t.Run("missingProfileFails", func(t *testing.T) {
		t.Parallel()

		out, err := runBashScript(t, root, script, nil, "80", filepath.Join(t.TempDir(), "missing.out"))
		if err == nil {
			t.Fatalf("expected error, got success:\n%s", out)
		}
		if !strings.Contains(out, "coverage profile not found") {
			t.Fatalf("expected missing profile error, got:\n%s", out)
		}
	})

	t.Run("thresholdEvaluation", func(t *testing.T) {
		t.Parallel()

		dir := t.TempDir()
		goFile := writeTempFile(t, dir, "sample-*.go", "package sample\n\nfunc Covered() {}\nfunc Uncovered() {}\n")
		profile := writeTempFile(t, dir, "coverage-*.out", "mode: set\n"+goFile+":3.1,3.17 1 1\n"+goFile+":4.1,4.19 1 0\n")

		cases := []struct {
			name      string
			threshold string
			wantOut   string
			wantErr   bool
		}{
			{name: "passesAt40", threshold: "40", wantOut: "total coverage:"},
			{name: "failsAt60", threshold: "60", wantErr: true, wantOut: "below required 60.0%"},
		}

		for _, tc := range cases {
			tc := tc
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()

				out, err := runBashScript(t, root, script, nil, tc.threshold, profile)
				if tc.wantErr {
					if err == nil {
						t.Fatalf("expected error, got success:\n%s", out)
					}
				} else if err != nil {
					t.Fatalf("unexpected error: %v\n%s", err, out)
				}
				if !strings.Contains(out, tc.wantOut) {
					t.Fatalf("expected output containing %q, got:\n%s", tc.wantOut, out)
				}
			})
		}
	})
}
