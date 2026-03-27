package labels

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestEmbeddedDataMatchesReinguardYAML(t *testing.T) {
	// Given: SSOT at .reinguard/labels.yaml and embedded copy for go:embed
	// When: comparing bytes
	// Then: they must match so runtime policy and tests agree
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	ssot := filepath.Join(root, ".reinguard", "labels.yaml")
	want, err := os.ReadFile(ssot)
	if err != nil {
		t.Fatalf("read %s: %v", ssot, err)
	}
	if !bytes.Equal(embeddedLabelsYAML, want) {
		t.Fatalf("internal/labels/data/labels.yaml must match .reinguard/labels.yaml byte-for-byte (copy SSOT into data/ for go:embed)")
	}
}

func TestTypeLabelsMatchEmbeddedYAML(t *testing.T) {
	// Given: parsed labels.yaml on disk and TypeLabels map from init()
	// When: comparing sorted type names
	// Then: they must match
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	yamlPath := filepath.Join(root, ".reinguard", "labels.yaml")
	data, err := os.ReadFile(yamlPath)
	if err != nil {
		t.Fatalf("read %s: %v", yamlPath, err)
	}
	fromFile, err := LoadFromBytes(data)
	if err != nil {
		t.Fatal(err)
	}
	fromFileNames := fromFile.TypeLabelNames()
	var fromMap []string
	for name := range TypeLabels {
		fromMap = append(fromMap, name)
	}
	slices.Sort(fromMap)
	if !slices.Equal(fromFileNames, fromMap) {
		t.Fatalf("labels.yaml type names %v != TypeLabels keys %v", fromFileNames, fromMap)
	}
}

func TestTypeLabelNamesMatchesTypeLabelsSet(t *testing.T) {
	names := TypeLabelNames()
	if len(names) != len(TypeLabels) {
		t.Fatalf("len(TypeLabelNames)=%d len(TypeLabels)=%d", len(names), len(TypeLabels))
	}
	for _, n := range names {
		if _, ok := TypeLabels[n]; !ok {
			t.Fatalf("TypeLabelNames contains %q not in TypeLabels", n)
		}
	}
}

func TestAllRepoLabelsTypeNamesMatchTypeLabels(t *testing.T) {
	c, err := LoadFromBytes(embeddedLabelsYAML)
	if err != nil {
		t.Fatal(err)
	}
	for _, spec := range c.AllRepoLabels() {
		switch spec.Name {
		case "hotfix", "no-issue", "epic":
			if _, ok := TypeLabels[spec.Name]; ok {
				t.Fatalf("non-type label %q must not be in TypeLabels", spec.Name)
			}
		default:
			if _, ok := TypeLabels[spec.Name]; !ok {
				t.Fatalf("AllRepoLabels has %q missing from TypeLabels (expected type)", spec.Name)
			}
		}
	}
}

func TestIssueTemplateTaskDropdownMatchesTypeLabels(t *testing.T) {
	// Given: task Issue Form YAML and TypeLabelNames() from SSOT
	// When: reading dropdown options from task.yml
	// Then: options equal TypeLabelNames() (sync script keeps them aligned)
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	taskPath := filepath.Join(root, ".github", "ISSUE_TEMPLATE", "task.yml")
	data, err := os.ReadFile(taskPath)
	if err != nil {
		t.Fatalf("read %s: %v", taskPath, err)
	}
	var doc struct {
		Body []struct {
			Type       string `yaml:"type"`
			Attributes struct {
				Options []string `yaml:"options"`
			} `yaml:"attributes"`
		} `yaml:"body"`
	}
	if err := yaml.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	if len(doc.Body) < 1 || doc.Body[0].Type != "dropdown" {
		t.Fatalf("task.yml: expected first body block to be type dropdown, got %#v", doc.Body)
	}
	got := doc.Body[0].Attributes.Options
	want := TypeLabelNames()
	if !slices.Equal(got, want) {
		t.Fatalf("task.yml Type options %v != labels TypeLabelNames %v (run .reinguard/scripts/sync-issue-templates.sh)", got, want)
	}
}

func TestParseTypeLabelsFromPRPolicy(t *testing.T) {
	sample := []byte(`foo
            const TYPE_LABELS = ['feat', 'fix', 'docs'];
            bar`)
	got, err := ParseTypeLabelsFromPRPolicy(sample)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"docs", "feat", "fix"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}
