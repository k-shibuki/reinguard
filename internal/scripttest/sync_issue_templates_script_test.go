package scripttest

import (
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestSyncIssueTemplatesScript_UpdatesTaskDropdown(t *testing.T) {
	// mustMikefarahYq uses t.Setenv, so this test must stay serial.
	root := repoRoot(t)
	mustMikefarahYq(t, root)
	script := scriptPath(t, "sync-issue-templates.sh")

	// Given: a temporary task template and explicit label-name JSON override.
	taskFile := writeTempFile(t, t.TempDir(), "task-*.yml", `body:
  - attributes:
      label: Type
      options:
        - old
`)
	env := []string{
		"REINGUARD_TASK_TEMPLATE_PATH=" + taskFile,
		`REINGUARD_LABELS_NAMES_JSON=["feat","fix","docs"]`,
	}

	// When: the sync script runs against the temporary template file.
	out, err := runBashScript(t, root, script, env)
	if err != nil {
		t.Fatalf("sync-issue-templates: %v\n%s", err, out)
	}

	// Then: the dropdown options are replaced with the provided label list.
	optionsOut, err := exec.Command("yq", "-r", ".body[0].attributes.options[]", taskFile).CombinedOutput()
	if err != nil {
		t.Fatalf("read synced options: %v\n%s", err, optionsOut)
	}
	gotOptions := strings.Fields(string(optionsOut))
	wantOptions := []string{"feat", "fix", "docs"}
	if !slices.Equal(gotOptions, wantOptions) {
		t.Fatalf("synced options = %v, want %v", gotOptions, wantOptions)
	}
}

func TestSyncIssueTemplatesScript_ResolvesRelativeOverrideFromRepoRoot(t *testing.T) {
	// mustMikefarahYq uses t.Setenv, so this test must stay serial.
	root := repoRoot(t)
	mustMikefarahYq(t, root)
	script := scriptPath(t, "sync-issue-templates.sh")

	// Given: a temporary repo root with a relative task-template override.
	taskDir := t.TempDir()
	taskFile := writeTempFile(t, taskDir, "task-*.yml", `body:
  - attributes:
      label: Type
      options:
        - old
`)
	relTask, err := filepath.Rel(root, taskFile)
	if err != nil {
		t.Fatal(err)
	}
	env := []string{
		"REINGUARD_TASK_TEMPLATE_PATH=" + relTask,
		`REINGUARD_LABELS_NAMES_JSON=["feat","fix"]`,
	}

	// When: the script runs from outside the repository root.
	out, err := runBashScript(t, taskDir, script, env)
	if err != nil {
		t.Fatalf("sync-issue-templates relative override: %v\n%s", err, out)
	}

	// Then: the relative override is resolved against the repository root, not the caller cwd.
	optionsOut, err := exec.Command("yq", "-r", ".body[0].attributes.options[]", taskFile).CombinedOutput()
	if err != nil {
		t.Fatalf("read synced options: %v\n%s", err, optionsOut)
	}
	gotOptions := strings.Fields(string(optionsOut))
	wantOptions := []string{"feat", "fix"}
	if !slices.Equal(gotOptions, wantOptions) {
		t.Fatalf("synced options = %v, want %v", gotOptions, wantOptions)
	}
}
