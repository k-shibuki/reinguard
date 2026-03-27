package labels

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// taskBodyMinimal is a valid Issue body for --template task (required sections).
const taskBodyMinimal = `## Context
Why.

## Refs: ADR
none

## ADR Impact
none

## Acceptance ↔ ADR
ok

## Definition of Done
- item

## Test plan
- go test ./...

## Linked issues
(none)
`

// epicBodyMinimal is a valid Issue body for --template epic.
const epicBodyMinimal = `## Summary
Epic summary.

## Background
Background.

## Verification baseline
go test ./...

## Child work items (sub-issues)
- #1
`

func mustMikefarahYq(t *testing.T, root string) {
	t.Helper()
	binDir := filepath.Join(root, ".reinguard", "scripts", ".bin")
	cached := filepath.Join(binDir, "yq")
	if fi, err := os.Stat(cached); err == nil && !fi.IsDir() && fi.Mode()&0111 != 0 {
		t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	}
	p, err := exec.LookPath("yq")
	if err != nil || p == "" {
		t.Skip("yq not in PATH (install mikefarah/yq v4 or run .reinguard/scripts/sync-issue-templates.sh once; see CONTRIBUTING.md)")
	}
	out, err := exec.Command("yq", "--version").CombinedOutput()
	if err != nil || !strings.Contains(string(out), "mikefarah") {
		t.Skip("mikefarah yq required for check-issue-policy.sh (see CONTRIBUTING.md)")
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

// TestCheckIssuePolicyScript runs .reinguard/scripts/check-issue-policy.sh against synthetic Issue metadata (template, labels, body).
func TestCheckIssuePolicyScript(t *testing.T) {
	root := repoRoot(t)
	mustMikefarahYq(t, root)
	script := filepath.Join(root, ".reinguard", "scripts", "check-issue-policy.sh")

	tests := []struct {
		name       string
		title      string
		template   string
		label      string
		body       string
		wantSubstr []string
		wantErr    bool
	}{
		{
			name:       "taskOK",
			title:      "feat(scope): add feature",
			template:   "task",
			label:      "feat",
			body:       taskBodyMinimal,
			wantSubstr: []string{"pre-flight OK"},
			wantErr:    false,
		},
		{
			name:       "taskBadTitle",
			title:      "not-a-conventional-title",
			template:   "task",
			label:      "feat",
			body:       taskBodyMinimal,
			wantSubstr: []string{"Issue title:", "Conventional Commits"},
			wantErr:    true,
		},
		{
			name:       "epicOK",
			title:      "epic: phase work",
			template:   "epic",
			label:      "epic",
			body:       epicBodyMinimal,
			wantSubstr: []string{"pre-flight OK"},
			wantErr:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: Issue body file and CLI args for check-issue-policy.sh
			f, err := os.CreateTemp(t.TempDir(), "issue-body-*.md")
			if err != nil {
				t.Fatal(err)
			}
			path := f.Name()
			if _, werr := f.WriteString(tt.body); werr != nil {
				t.Fatal(werr)
			}
			if cerr := f.Close(); cerr != nil {
				t.Fatal(cerr)
			}

			// When: running the script
			cmd := exec.Command("bash", script, "--title", tt.title, "--body-file", path, "--label", tt.label, "--template", tt.template)
			cmd.Dir = root
			out, err := cmd.CombinedOutput()
			s := string(out)

			// Then: exit code and output match expectations
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected non-zero exit, got success: %s", s)
				}
			} else {
				if err != nil {
					t.Fatalf("check-issue-policy: %v\n%s", err, s)
				}
			}
			for _, sub := range tt.wantSubstr {
				if !strings.Contains(s, sub) {
					t.Fatalf("expected output to contain %q, got:\n%s", sub, s)
				}
			}
		})
	}
}
