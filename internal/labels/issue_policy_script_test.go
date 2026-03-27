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

func TestCheckIssuePolicyScript_taskOK(t *testing.T) {
	// Given: valid task template body and feat title/label
	// When: running check-issue-policy.sh
	// Then: exit 0 and output contains OK
	root := repoRoot(t)
	mustMikefarahYq(t, root)
	script := filepath.Join(root, ".reinguard", "scripts", "check-issue-policy.sh")
	f, err := os.CreateTemp(t.TempDir(), "issue-body-*.md")
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()
	if _, werr := f.WriteString(taskBodyMinimal); werr != nil {
		t.Fatal(werr)
	}
	if cerr := f.Close(); cerr != nil {
		t.Fatal(cerr)
	}
	cmd := exec.Command("bash", script, "--title", "feat(scope): add feature", "--body-file", path, "--label", "feat", "--template", "task")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("check-issue-policy: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "OK") {
		t.Fatalf("expected OK in output: %s", out)
	}
}

func TestCheckIssuePolicyScript_taskBadTitle(t *testing.T) {
	// Given: valid body but non-Conventional Commits title
	// When: running check-issue-policy.sh
	// Then: non-zero exit
	root := repoRoot(t)
	mustMikefarahYq(t, root)
	script := filepath.Join(root, ".reinguard", "scripts", "check-issue-policy.sh")
	f, err := os.CreateTemp(t.TempDir(), "issue-body-*.md")
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()
	if _, werr := f.WriteString(taskBodyMinimal); werr != nil {
		t.Fatal(werr)
	}
	if cerr := f.Close(); cerr != nil {
		t.Fatal(cerr)
	}
	cmd := exec.Command("bash", script, "--title", "not-a-conventional-title", "--body-file", path, "--label", "feat", "--template", "task")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err == nil {
		t.Fatalf("expected non-zero exit, got success: %s", out)
	}
}

func TestCheckIssuePolicyScript_epicOK(t *testing.T) {
	// Given: epic template body and epic label
	// When: running check-issue-policy.sh
	// Then: exit 0
	root := repoRoot(t)
	mustMikefarahYq(t, root)
	script := filepath.Join(root, ".reinguard", "scripts", "check-issue-policy.sh")
	f, err := os.CreateTemp(t.TempDir(), "epic-body-*.md")
	if err != nil {
		t.Fatal(err)
	}
	path := f.Name()
	if _, werr := f.WriteString(epicBodyMinimal); werr != nil {
		t.Fatal(werr)
	}
	if cerr := f.Close(); cerr != nil {
		t.Fatal(cerr)
	}
	cmd := exec.Command("bash", script, "--title", "epic: phase work", "--body-file", path, "--label", "epic", "--template", "epic")
	cmd.Dir = root
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("check-issue-policy: %v\n%s", err, out)
	}
}
