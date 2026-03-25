package rgdcli

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/urfave/cli/v2"
)

func TestRunObserve_gitOnlyProvider(t *testing.T) {
	t.Parallel()
	root := newObserveGitRepo(t)

	var buf bytes.Buffer
	app := testApp(t, "t")
	app.Writer = &buf
	if err := app.Run([]string{"rgd", "observe", "git", "--cwd", root}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"git"`)) || !bytes.Contains(buf.Bytes(), []byte(`"signals"`)) {
		t.Fatalf("%s", buf.String())
	}
}

func TestRunObserve_commandMatrix(t *testing.T) {
	t.Parallel()
	root := newObserveGitRepo(t)

	cases := []struct {
		name    string
		prefix  []string
		wantGit bool
	}{
		{name: "observe", prefix: []string{"observe"}, wantGit: true},
		{name: "workflow-position", prefix: []string{"observe", "workflow-position"}, wantGit: true},
		{name: "git", prefix: []string{"observe", "git"}, wantGit: true},
		{name: "github", prefix: []string{"observe", "github"}, wantGit: false},
		{name: "github-issues", prefix: []string{"observe", "github", "issues"}, wantGit: false},
		{name: "github-pull-requests", prefix: []string{"observe", "github", "pull-requests"}, wantGit: false},
		{name: "github-ci", prefix: []string{"observe", "github", "ci"}, wantGit: false},
		{name: "github-reviews", prefix: []string{"observe", "github", "reviews"}, wantGit: false},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			args := []string{"rgd"}
			args = append(args, tc.prefix...)
			args = append(args, "--cwd", root)

			var buf bytes.Buffer
			app := testApp(t, "t")
			app.Writer = &buf
			if err := app.Run(args); err != nil {
				t.Fatal(err)
			}
			var doc map[string]any
			if err := json.Unmarshal(buf.Bytes(), &doc); err != nil {
				t.Fatalf("invalid JSON: %v\n%s", err, buf.String())
			}
			if doc["schema_version"] == nil {
				t.Fatalf("missing schema_version: %s", buf.String())
			}
			if !tc.wantGit {
				return
			}
			signals, _ := doc["signals"].(map[string]any)
			if _, ok := signals["git"]; !ok {
				t.Fatalf("expected git signals, got: %s", buf.String())
			}
		})
	}
}

func TestRunObserve_missingConfig(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	app := testApp(t, "t")
	err := app.Run([]string{"rgd", "observe", "git", "--cwd", root})
	if err == nil {
		t.Fatal("expected error when .reinguard is missing")
	}
}

func TestRunObserve_failOnNonResolved_degraded(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	runGitForObserve(t, root, "init")
	runGitForObserve(t, root, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init")
	cfg := filepath.Join(root, ".reinguard")
	if err := os.MkdirAll(cfg, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(cfg, "reinguard.yaml"), []byte(testFixtureReinguardUnknownProvider))
	if err := os.MkdirAll(filepath.Join(cfg, "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(cfg, "rules", "d.yaml"), []byte(testFixtureRulesEmpty))

	var buf bytes.Buffer
	app := testApp(t, "t")
	app.Writer = &buf
	err := app.Run([]string{"rgd", "observe", "--cwd", root, "--fail-on-non-resolved"})
	var exitErr cli.ExitCoder
	if !errors.As(err, &exitErr) || exitErr.ExitCode() != 1 {
		t.Fatalf("want exit code 1, got err=%v", err)
	}
}

func newObserveGitRepo(t *testing.T) string {
	t.Helper()
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
	return root
}

func runGitForObserve(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v %s", args, err, string(out))
	}
}
