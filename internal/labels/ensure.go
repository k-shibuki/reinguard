package labels

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sort"
)

// RepoLabel is a label to ensure exists on the current gh repo (idempotent).
type RepoLabel struct {
	Name        string
	Color       string
	Description string
}

// policyRepoLabels returns type + exception labels (same contract as former tools/gh-labels.sh).
func policyRepoLabels() []RepoLabel {
	typeSpecs := []RepoLabel{
		{"feat", "A2EEEF", "Type: new feature"},
		{"fix", "D73A4A", "Type: bug fix"},
		{"refactor", "FEF2C0", "Type: refactor"},
		{"perf", "5319E7", "Type: performance"},
		{"docs", "0075CA", "Type: documentation"},
		{"test", "0E8A16", "Type: tests"},
		{"ci", "FBCA04", "Type: CI config"},
		{"build", "D4C5F9", "Type: build system"},
		{"chore", "C2E0C6", "Type: chore"},
		{"style", "BFDADC", "Type: formatting / style"},
		{"revert", "D4C5F9", "Type: revert"},
	}
	exc := []RepoLabel{
		{"hotfix", "B60205", "Exception: urgent fix without normal Issue flow"},
		{"no-issue", "F9D0C4", "Exception: PR without linked Issue (justification required)"},
	}
	return append(typeSpecs, exc...)
}

type ghLabelRow struct {
	Name string `json:"name"`
}

// EnsureRepoLabels creates missing labels via gh (requires gh authenticated for the repo).
func EnsureRepoLabels(w io.Writer) error {
	existing, err := listLabelNames()
	if err != nil {
		return err
	}
	for _, spec := range policyRepoLabels() {
		if existing[spec.Name] {
			_, _ = fmt.Fprintf(w, "exists: %s\n", spec.Name)
			continue
		}
		if err := createLabel(spec); err != nil {
			return err
		}
		_, _ = fmt.Fprintf(w, "created: %s\n", spec.Name)
	}
	_, _ = fmt.Fprintln(w, "Done.")
	return nil
}

func listLabelNames() (map[string]bool, error) {
	cmd := exec.Command("gh", "label", "list", "--json", "name")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("gh label list: %w\n%s", err, stderr.String())
	}
	var rows []ghLabelRow
	if err := json.Unmarshal(out, &rows); err != nil {
		return nil, fmt.Errorf("parse gh label list json: %w", err)
	}
	m := make(map[string]bool, len(rows))
	for _, r := range rows {
		m[r.Name] = true
	}
	return m, nil
}

func createLabel(spec RepoLabel) error {
	cmd := exec.Command("gh", "label", "create", spec.Name,
		"--color", spec.Color, "--description", spec.Description)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh label create %q: %w\n%s", spec.Name, err, stderr.String())
	}
	return nil
}

// ParseTypeLabelsFromPRPolicy extracts TYPE_LABELS from pr-policy.yaml workflow source (for tests).
func ParseTypeLabelsFromPRPolicy(yaml []byte) ([]string, error) {
	// Match: const TYPE_LABELS = ['feat', 'fix', ...];
	const prefix = "const TYPE_LABELS = ["
	i := bytes.Index(yaml, []byte(prefix))
	if i < 0 {
		return nil, fmt.Errorf("TYPE_LABELS array not found in pr-policy workflow")
	}
	i += len(prefix)
	j := bytes.Index(yaml[i:], []byte("];"))
	if j < 0 {
		return nil, fmt.Errorf("TYPE_LABELS array terminator not found")
	}
	inner := yaml[i : i+j]
	var names []string
	for _, part := range bytes.Split(inner, []byte{','}) {
		p := bytes.TrimSpace(part)
		if len(p) >= 2 && p[0] == '\'' && p[len(p)-1] == '\'' {
			names = append(names, string(p[1:len(p)-1]))
		}
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("no type labels parsed from TYPE_LABELS")
	}
	sort.Strings(names)
	return names, nil
}
