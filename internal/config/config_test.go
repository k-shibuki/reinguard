package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoad_emptyConfigDir(t *testing.T) {
	t.Parallel()
	_, err := Load("")
	if err == nil || !strings.Contains(err.Error(), "empty directory") {
		t.Fatalf("got %v", err)
	}
}

func TestDeprecatedWarnings_nilRoot(t *testing.T) {
	t.Parallel()
	if w := DeprecatedWarnings(nil); len(w) != 0 {
		t.Fatalf("got %v", w)
	}
}

func TestDeprecatedWarnings_noLegacyHints(t *testing.T) {
	t.Parallel()
	r := &Root{SchemaVersion: "0.2.0", DefaultBranch: "main"}
	if w := DeprecatedWarnings(r); len(w) != 0 {
		t.Fatalf("got %v", w)
	}
}

func TestDeprecatedWarnings_legacyHints(t *testing.T) {
	t.Parallel()
	r := &Root{LegacyToolHints: map[string]any{"x": 1}}
	w := DeprecatedWarnings(r)
	if len(w) != 1 || !strings.Contains(w[0], "legacy_tool_hints") {
		t.Fatalf("got %v", w)
	}
}

func TestLoad_knowledgeManifest_ok(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), []byte(`schema_version: "0.2.0"
default_branch: main
providers: []
`))
	if err := os.Mkdir(filepath.Join(dir, "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "knowledge"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "knowledge", "manifest.json"), []byte(`{
  "schema_version": "0.1.0",
  "entries": [{"id": "doc1", "path": "docs/a.md"}]
}`))

	res, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !res.KnowledgePresent || res.Knowledge == nil {
		t.Fatal("expected knowledge manifest")
	}
	if len(res.Knowledge.Entries) != 1 || res.Knowledge.Entries[0].ID != "doc1" {
		t.Fatalf("%+v", res.Knowledge)
	}
}

func TestLoad_knowledgeManifest_invalidJSON(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), []byte(`schema_version: "0.2.0"
default_branch: main
providers: []
`))
	if err := os.Mkdir(filepath.Join(dir, "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "knowledge"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "knowledge", "manifest.json"), []byte(`{`))

	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "parse knowledge manifest") {
		t.Fatalf("got %v", err)
	}
}

func TestLoad_knowledgeManifest_schemaInvalid(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), []byte(`schema_version: "0.2.0"
default_branch: main
providers: []
`))
	if err := os.Mkdir(filepath.Join(dir, "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, "knowledge"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Missing required "entries"
	writeFile(t, filepath.Join(dir, "knowledge", "manifest.json"), []byte(`{"schema_version":"0.1.0"}`))

	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "schema validation") {
		t.Fatalf("got %v", err)
	}
}

func TestLoad_rulesDotYmlAndStableOrder(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), []byte(`schema_version: "0.2.0"
default_branch: main
providers: []
`))
	if err := os.Mkdir(filepath.Join(dir, "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "rules", "z.yml"), []byte(`rules:
  - type: state
    id: z
    priority: 10
    when:
      op: eq
    state_id: Z
`))
	writeFile(t, filepath.Join(dir, "rules", "a.yaml"), []byte(`rules:
  - type: state
    id: a
    priority: 10
    when:
      op: eq
    state_id: A
`))
	if err := os.Mkdir(filepath.Join(dir, "rules", "skipdir"), 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	names := []string{}
	for n := range res.RuleFiles {
		names = append(names, n)
	}
	if len(names) != 2 {
		t.Fatalf("files: %v", names)
	}
	rules := res.Rules()
	if len(rules) != 2 || rules[0].ID != "a" || rules[1].ID != "z" {
		t.Fatalf("expected [a, z] ordering, got %+v", rules)
	}
}

func TestLoad_minimalValid(t *testing.T) {
	t.Parallel()
	// Given: valid reinguard.yaml and empty rules directory
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), []byte(`schema_version: "0.2.0"
default_branch: main
providers:
  - id: git
    enabled: true
`))
	if err := os.Mkdir(filepath.Join(dir, "rules"), 0o755); err != nil {
		t.Fatal(err)
	}

	// When: Load is called
	res, err := Load(dir)

	// Then: No error and schema version matches
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if res.Root.SchemaVersion != "0.2.0" {
		t.Fatalf("schema_version: got %q", res.Root.SchemaVersion)
	}
	if res.Root.DefaultBranch != "main" {
		t.Fatalf("default_branch: got %q", res.Root.DefaultBranch)
	}
}

func TestLoad_missingDefaultBranch(t *testing.T) {
	t.Parallel()
	// Given: root config missing required default_branch
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), []byte(`schema_version: "0.2.0"
providers: []
`))

	// When: Load is called
	_, err := Load(dir)

	// Then: schema validation mentions the failure
	if err == nil {
		t.Fatal("expected error for missing default_branch")
	}
	if !strings.Contains(err.Error(), "schema validation") {
		t.Fatal(err)
	}
}

func TestLoad_brokenYAML(t *testing.T) {
	t.Parallel()
	// Given: invalid YAML in reinguard.yaml
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), []byte("schema_version: [\n"))

	// When: Load is called
	_, err := Load(dir)

	// Then: parse error references file
	if err == nil {
		t.Fatal("expected parse error")
	}
	if !strings.Contains(err.Error(), "reinguard.yaml") {
		t.Fatal(err)
	}
}

func TestLoad_missingConfigFile(t *testing.T) {
	t.Parallel()
	// Given: directory without reinguard.yaml
	dir := t.TempDir()

	// When: Load is called
	_, err := Load(dir)

	// Then: read error
	if err == nil || !strings.Contains(err.Error(), "reinguard.yaml") {
		t.Fatalf("%v", err)
	}
}

func TestLoad_duplicateProviderID(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), []byte(`schema_version: "0.2.0"
default_branch: main
providers:
  - id: git
    enabled: true
  - id: git
    enabled: false
`))

	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "duplicate provider id") {
		t.Fatalf("got err=%v", err)
	}
}

func TestLoad_rulesFile(t *testing.T) {
	t.Parallel()
	// Given: valid root and one rules file
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), []byte(`schema_version: "0.2.0"
default_branch: main
providers: []
`))
	if err := os.Mkdir(filepath.Join(dir, "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	rulePath := filepath.Join(dir, "rules", "a.yaml")
	writeFile(t, rulePath, []byte(`rules:
  - type: state
    id: s1
    priority: 10
    when:
      op: eq
    state_id: Idle
`))

	// When: Load is called
	res, err := Load(dir)

	// Then: rule is present
	if err != nil {
		t.Fatal(err)
	}
	rs := res.Rules()
	if len(rs) != 1 || rs[0].ID != "s1" {
		t.Fatalf("rules: %+v", rs)
	}
}

func TestLoad_rulesSchemaInvalid(t *testing.T) {
	t.Parallel()
	// Given: rules file missing required when shape (empty when object may still parse — use missing rules key)
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), []byte(`schema_version: "0.2.0"
default_branch: main
providers: []
`))
	if err := os.Mkdir(filepath.Join(dir, "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "rules", "bad.yaml"), []byte(`rules: not-an-array
`))

	// When: Load is called
	_, err := Load(dir)

	// Then: validation error names rules file
	if err == nil || !strings.Contains(err.Error(), "bad.yaml") || !strings.Contains(err.Error(), "schema validation") {
		t.Fatalf("%v", err)
	}
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
