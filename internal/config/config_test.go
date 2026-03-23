package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
