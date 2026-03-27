package labels

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os/exec"
	"sort"
	"strings"
)

// ghLabelRow is a subset of gh label list --json output.
type ghLabelRow struct {
	Name        string `json:"name"`
	Color       string `json:"color"`
	Description string `json:"description"`
}

// EnsureRepoLabels creates missing labels via gh (requires gh authenticated for the repo).
func EnsureRepoLabels(w io.Writer, configDir string) error {
	cfg, err := LoadFromConfigDir(configDir)
	if err != nil {
		return err
	}
	existing, err := listLabelNames()
	if err != nil {
		return err
	}
	for _, spec := range cfg.AllRepoLabels() {
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

// SyncRepoLabels syncs labels.yaml to GitHub: create missing, update color/description when different.
// If dryRun, prints planned actions to w and does not mutate GitHub.
func SyncRepoLabels(w io.Writer, configDir string, dryRun bool) error {
	cfg, err := LoadFromConfigDir(configDir)
	if err != nil {
		return err
	}
	want := cfg.LabelsByName()
	rows, err := listLabelRows()
	if err != nil {
		return err
	}
	have := make(map[string]ghLabelRow, len(rows))
	for _, r := range rows {
		have[r.Name] = r
	}
	keys := make([]string, 0, len(want))
	for k := range want {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, name := range keys {
		entry := want[name]
		spec := RepoLabel{
			Name:        entry.Name,
			Color:       strings.TrimPrefix(strings.ToUpper(entry.Color), "#"),
			Description: entry.Description,
		}
		cur, ok := have[name]
		if !ok {
			if dryRun {
				_, _ = fmt.Fprintf(w, "[dry-run] would create %s color=%s\n", spec.Name, spec.Color)
				continue
			}
			if err := createLabel(spec); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(w, "created: %s\n", spec.Name)
			continue
		}
		curColor := normalizeGHColor(cur.Color)
		if curColor != spec.Color || cur.Description != spec.Description {
			if dryRun {
				_, _ = fmt.Fprintf(w, "[dry-run] would edit %s (%s -> %s)\n", name, curColor, spec.Color)
				continue
			}
			if err := editLabel(spec); err != nil {
				return err
			}
			_, _ = fmt.Fprintf(w, "updated: %s\n", spec.Name)
		}
	}
	if dryRun {
		_, _ = fmt.Fprintln(w, "Done (dry-run).")
	} else {
		_, _ = fmt.Fprintln(w, "Done.")
	}
	return nil
}

func normalizeGHColor(s string) string {
	return strings.TrimPrefix(strings.ToUpper(strings.TrimSpace(s)), "#")
}

func listLabelRows() ([]ghLabelRow, error) {
	cmd := exec.Command("gh", "label", "list", "--json", "name,color,description")
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
	return rows, nil
}

func listLabelNames() (map[string]bool, error) {
	rows, err := listLabelRows()
	if err != nil {
		return nil, err
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

func editLabel(spec RepoLabel) error {
	cmd := exec.Command("gh", "label", "edit", spec.Name,
		"--color", spec.Color, "--description", spec.Description)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("gh label edit %q: %w\n%s", spec.Name, err, stderr.String())
	}
	return nil
}

// ParseTypeLabelsFromPRPolicy extracts TYPE_LABELS from PR policy script source (for tests).
// Expected in .github/scripts/pr-policy-check.js: const TYPE_LABELS = [ "feat", ... ];
func ParseTypeLabelsFromPRPolicy(src []byte) ([]string, error) {
	const prefix = "const TYPE_LABELS = ["
	i := bytes.Index(src, []byte(prefix))
	if i < 0 {
		return nil, fmt.Errorf("TYPE_LABELS array not found in PR policy script")
	}
	i += len(prefix)
	j := bytes.Index(src[i:], []byte("];"))
	if j < 0 {
		return nil, fmt.Errorf("TYPE_LABELS array terminator not found")
	}
	inner := src[i : i+j]
	var names []string
	for _, part := range bytes.Split(inner, []byte{','}) {
		p := bytes.TrimSpace(part)
		if len(p) >= 2 {
			if p[0] == '\'' && p[len(p)-1] == '\'' {
				names = append(names, string(p[1:len(p)-1]))
				continue
			}
			if p[0] == '"' && p[len(p)-1] == '"' {
				names = append(names, string(p[1:len(p)-1]))
			}
		}
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("no type labels parsed from TYPE_LABELS")
	}
	sort.Strings(names)
	return names, nil
}
