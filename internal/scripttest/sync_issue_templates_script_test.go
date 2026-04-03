package scripttest

import (
	"os"
	"strings"
	"testing"
)

func TestSyncIssueTemplatesScript_UpdatesTaskDropdown(t *testing.T) {
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
	updated, err := os.ReadFile(taskFile)
	if err != nil {
		t.Fatal(err)
	}
	got := string(updated)
	for _, want := range []string{"feat", "fix", "docs"} {
		if !strings.Contains(got, want) {
			t.Fatalf("expected task template to contain %q, got:\n%s", want, got)
		}
	}
	if strings.Contains(got, "- old") {
		t.Fatalf("expected old option to be replaced, got:\n%s", got)
	}
}
