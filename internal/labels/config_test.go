package labels

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestLoadFromBytes_invalidYAML(t *testing.T) {
	// Given: invalid YAML
	// When: LoadFromBytes runs
	// Then: parse error is returned
	_, err := LoadFromBytes([]byte(":\n"))
	if err == nil || !strings.Contains(err.Error(), "parse yaml") {
		t.Fatalf("expected parse yaml error, got %v", err)
	}
}

func TestLoadFromFile_and_LoadFromConfigDir(t *testing.T) {
	// Given: a directory containing labels.yaml
	// When: LoadFromFile and LoadFromConfigDir are called
	// Then: config loads; missing file returns error
	dir := t.TempDir()
	path := filepath.Join(dir, "labels.yaml")
	yaml := `schema_version: "0.1.0"
categories:
  type:
    scope: shared
    labels:
      - name: feat
        color: "#A2EEEF"
        description: "t"
`
	if err := os.WriteFile(path, []byte(yaml), 0o600); err != nil {
		t.Fatal(err)
	}
	c, err := LoadFromFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if got := c.TypeLabelNames(); !slices.Equal(got, []string{"feat"}) {
		t.Fatalf("TypeLabelNames: %v", got)
	}
	c2, err := LoadFromConfigDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if c2.SchemaVersion != "0.1.0" {
		t.Fatalf("schema: %q", c2.SchemaVersion)
	}
	if _, err := LoadFromFile(filepath.Join(dir, "nope.yaml")); err == nil {
		t.Fatal("expected error for missing file")
	}
}

func TestConfig_TypeLabelNames_edgeCases(t *testing.T) {
	// Given: nil Config or type category with non-shared scope
	// When: TypeLabelNames is called
	// Then: nil is returned
	if names := ((*Config)(nil)).TypeLabelNames(); names != nil {
		t.Fatalf("nil config: %v", names)
	}
	c := &Config{Categories: map[string]Category{
		"type": {Scope: "issue", Labels: []LabelEntry{{Name: "feat"}}},
	}}
	if got := c.TypeLabelNames(); got != nil {
		t.Fatalf("wrong scope: %v", got)
	}
}

func TestConfig_ExceptionIssueCommit_LabelsByName(t *testing.T) {
	// Given: YAML with type, exception, and scope categories
	// When: accessors run
	// Then: names and labels-by-name match expectations
	yaml := `schema_version: "0.1.0"
categories:
  type:
    scope: shared
    labels:
      - name: feat
        color: "A2EEEF"
        description: "t"
        commit_prefix: true
  exception:
    labels:
      - name: hotfix
        color: "B60205"
        description: "e"
        commit_prefix: true
  scope:
    labels:
      - name: epic
        color: "7057FF"
        description: "s"
`
	c, err := LoadFromBytes([]byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	if got := c.ExceptionLabelNames(); !slices.Equal(got, []string{"hotfix"}) {
		t.Fatalf("ExceptionLabelNames: %v", got)
	}
	if got := c.IssueOnlyLabelNames(); !slices.Equal(got, []string{"epic"}) {
		t.Fatalf("IssueOnlyLabelNames: %v", got)
	}
	cp := c.CommitPrefixNames()
	if !slices.Contains(cp, "feat") || !slices.Contains(cp, "hotfix") {
		t.Fatalf("CommitPrefixNames: %v", cp)
	}
	by := c.LabelsByName()
	if by["feat"].Name != "feat" || by["epic"].Name != "epic" {
		t.Fatalf("LabelsByName: %#v", by)
	}
}

func TestConfig_missingExceptionAndScopeCategories(t *testing.T) {
	// Given: Config with only type category (no exception / scope keys)
	// When: ExceptionLabelNames and IssueOnlyLabelNames are called
	// Then: nil (category missing)
	yaml := `categories:
  type:
    scope: shared
    labels:
      - name: feat
        color: "A2EEEF"
        description: "t"
`
	c, err := LoadFromBytes([]byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	if got := c.ExceptionLabelNames(); got != nil {
		t.Fatalf("ExceptionLabelNames: %v", got)
	}
	if got := c.IssueOnlyLabelNames(); got != nil {
		t.Fatalf("IssueOnlyLabelNames: %v", got)
	}
}

func TestConfig_LabelsByName_duplicateLastWins(t *testing.T) {
	// Given: duplicate label names in one category
	// When: LabelsByName is called
	// Then: last entry wins
	yaml := `categories:
  only:
    labels:
      - name: dup
        color: "111111"
        description: first
      - name: dup
        color: "222222"
        description: second
`
	c, err := LoadFromBytes([]byte(yaml))
	if err != nil {
		t.Fatal(err)
	}
	if c.LabelsByName()["dup"].Description != "second" {
		t.Fatalf("expected last wins")
	}
}

func TestConfig_nilCategories(t *testing.T) {
	// Given: empty Config (no Categories)
	// When: category helpers run
	// Then: nil or empty-appropriate results
	c := &Config{}
	if c.ExceptionLabelNames() != nil {
		t.Fatal()
	}
	if c.IssueOnlyLabelNames() != nil {
		t.Fatal()
	}
	if c.CommitPrefixNames() != nil {
		t.Fatal()
	}
	if c.LabelsByName() != nil {
		t.Fatal()
	}
	if c.AllRepoLabels() != nil {
		t.Fatal()
	}
}

func TestNormalizeGHColor(t *testing.T) {
	t.Parallel()
	// Given: GitHub color strings with varied spacing and # prefix
	// When: normalizeGHColor runs per row
	// Then: uppercase hex without # matches want
	tests := []struct {
		name string
		in   string
		want string
	}{
		{"trim_and_hash_prefix", "  #aabbcc  ", "AABBCC"},
		{"hash_prefix", "#DEADBEEF", "DEADBEEF"},
		{"no_hash", "00ff00", "00FF00"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			if got := normalizeGHColor(tc.in); got != tc.want {
				t.Fatalf("normalizeGHColor(%q)=%q want %q", tc.in, got, tc.want)
			}
		})
	}
}

func TestParseTypeLabelsFromPRPolicy_errors(t *testing.T) {
	// Given: invalid PR policy script fragments
	// When: ParseTypeLabelsFromPRPolicy runs
	// Then: each case returns an error
	cases := []struct {
		name string
		src  []byte
	}{
		{"no_array", []byte("no array")},
		{"unterminated_array", []byte("const TYPE_LABELS = ['x'")},
		{"empty_labels", []byte("const TYPE_LABELS = [];")},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if _, err := ParseTypeLabelsFromPRPolicy(tc.src); err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

func TestParseTypeLabelsFromPRPolicy_doubleQuotes(t *testing.T) {
	// Given: TYPE_LABELS with double-quoted entries
	// When: parsing
	// Then: sorted names returned
	src := []byte(`const TYPE_LABELS = [ "feat", "fix" ];`)
	got, err := ParseTypeLabelsFromPRPolicy(src)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"feat", "fix"}
	if !slices.Equal(got, want) {
		t.Fatalf("got %v want %v", got, want)
	}
}
