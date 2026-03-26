package labels

import (
	"os"
	"path/filepath"
	"runtime"
	"slices"
	"testing"
)

func TestTypeLabelsMatchPRPolicyWorkflow(t *testing.T) {
	// Given: pr-policy-check.js TYPE_LABELS and labels.TypeLabels in Go
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	policyPath := filepath.Join(root, ".github", "scripts", "pr-policy-check.js")
	data, err := os.ReadFile(policyPath)
	if err != nil {
		t.Fatalf("read %s: %v", policyPath, err)
	}
	// When: script labels are parsed and compared to map keys
	fromScript, err := ParseTypeLabelsFromPRPolicy(data)
	if err != nil {
		t.Fatal(err)
	}
	var fromGo []string
	for name := range TypeLabels {
		fromGo = append(fromGo, name)
	}
	slices.Sort(fromGo)
	// Then: sorted lists match exactly
	if !slices.Equal(fromScript, fromGo) {
		t.Fatalf("TYPE_LABELS in pr-policy-check.js %v != labels.TypeLabels keys %v", fromScript, fromGo)
	}
}

func TestTypeLabelNamesMatchesTypeLabelsSet(t *testing.T) {
	// Given: TypeLabelNames slice and TypeLabels map
	names := TypeLabelNames()
	// When/Then: every name exists in the map and counts match
	if len(names) != len(TypeLabels) {
		t.Fatalf("len(TypeLabelNames)=%d len(TypeLabels)=%d", len(names), len(TypeLabels))
	}
	for _, n := range names {
		if _, ok := TypeLabels[n]; !ok {
			t.Fatalf("TypeLabelNames contains %q not in TypeLabels", n)
		}
	}
}

func TestPolicyRepoLabelsTypeNamesMatchTypeLabels(t *testing.T) {
	// Given: policyRepoLabels catalog and TypeLabels set
	// When/Then: each policy type label (except exceptions) is in TypeLabels
	for _, spec := range policyRepoLabels() {
		switch spec.Name {
		case "hotfix", "no-issue":
			if _, ok := TypeLabels[spec.Name]; ok {
				t.Fatalf("exception label %q must not be in TypeLabels", spec.Name)
			}
		default:
			if _, ok := TypeLabels[spec.Name]; !ok {
				t.Fatalf("policyRepoLabels has %q missing from TypeLabels", spec.Name)
			}
		}
	}
}

func TestParseTypeLabelsFromPRPolicy(t *testing.T) {
	// Given: JS source containing TYPE_LABELS array
	sample := []byte(`foo
            const TYPE_LABELS = ['feat', 'fix', 'docs'];
            bar`)
	// When: ParseTypeLabelsFromPRPolicy runs
	got, err := ParseTypeLabelsFromPRPolicy(sample)
	if err != nil {
		t.Fatal(err)
	}
	// Then: labels are returned sorted
	want := []string{"docs", "feat", "fix"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}
