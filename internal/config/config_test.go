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
	r := &Root{SchemaVersion: "0.3.0", DefaultBranch: "main"}
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
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), []byte(`schema_version: "0.3.0"
default_branch: main
providers: []
`))
	if err := os.Mkdir(filepath.Join(dir, "knowledge"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "knowledge", "manifest.json"), []byte(`{
  "schema_version": "0.3.0",
  "entries": [{
    "id": "doc1",
    "path": "docs/a.md",
    "description": "test doc",
    "triggers": ["test"]
  }]
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
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), []byte(`schema_version: "0.3.0"
default_branch: main
providers: []
`))
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
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), []byte(`schema_version: "0.3.0"
default_branch: main
providers: []
`))
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

func TestLoad_controlStatesDotYmlAndStableOrder(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), []byte(`schema_version: "0.3.0"
default_branch: main
providers: []
`))
	statesDir := filepath.Join(dir, "control", "states")
	if err := os.MkdirAll(statesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(statesDir, "z.yml"), []byte(`rules:
  - type: state
    id: z
    priority: 10
    when:
      op: eq
    state_id: Z
`))
	writeFile(t, filepath.Join(statesDir, "a.yaml"), []byte(`rules:
  - type: state
    id: a
    priority: 10
    when:
      op: eq
    state_id: A
`))
	if err := os.Mkdir(filepath.Join(statesDir, "skipdir"), 0o755); err != nil {
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

func TestLoad_legacyRulesYAML_rejected(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), []byte(`schema_version: "0.3.0"
default_branch: main
providers: []
`))
	legacyDir := filepath.Join(dir, "rules")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(legacyDir, "old.yaml"), []byte("rules: []\n"))

	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "legacy rules") {
		t.Fatalf("got %v", err)
	}
}

func TestLoad_minimalValid(t *testing.T) {
	t.Parallel()
	// Given: valid reinguard.yaml and no control/ directory
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), []byte(`schema_version: "0.3.0"
default_branch: main
providers:
  - id: git
    enabled: true
`))

	// When: Load is called
	res, err := Load(dir)

	// Then: No error and schema version matches
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if res.Root.SchemaVersion != "0.3.0" {
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
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), []byte(`schema_version: "0.3.0"
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
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), []byte(`schema_version: "0.3.0"
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

func TestLoad_controlStatesFile(t *testing.T) {
	t.Parallel()
	// Given: valid root and one state rules file
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), []byte(`schema_version: "0.3.0"
default_branch: main
providers: []
`))
	statesDir := filepath.Join(dir, "control", "states")
	if err := os.MkdirAll(statesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	rulePath := filepath.Join(statesDir, "a.yaml")
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

func TestLoad_controlStatesSchemaInvalid(t *testing.T) {
	t.Parallel()
	// Given: rules file missing required when shape (empty when object may still parse — use missing rules key)
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), []byte(`schema_version: "0.3.0"
default_branch: main
providers: []
`))
	statesDir := filepath.Join(dir, "control", "states")
	if err := os.MkdirAll(statesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(statesDir, "bad.yaml"), []byte(`rules: not-an-array
`))

	// When: Load is called
	_, err := Load(dir)

	// Then: validation error names rules file
	if err == nil || !strings.Contains(err.Error(), "bad.yaml") || !strings.Contains(err.Error(), "schema validation") {
		t.Fatalf("%v", err)
	}
}

func TestLoad_controlKindTypeMismatchRejected(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name     string
		kind     string
		yamlBody string
		wantSub  string
	}{
		{
			name: "states_dir_has_route_type",
			kind: "states",
			yamlBody: `rules:
  - type: route
    id: r1
    priority: 10
    route_id: next
    when: {op: eq, path: x, value: 1}
`,
			wantSub: `expected "state"`,
		},
		{
			name: "routes_dir_has_state_type",
			kind: "routes",
			yamlBody: `rules:
  - type: state
    id: s1
    priority: 10
    state_id: idle
    when: {op: eq, path: x, value: 1}
`,
			wantSub: `expected "route"`,
		},
		{
			name: "guards_dir_has_state_type",
			kind: "guards",
			yamlBody: `rules:
  - type: state
    id: s1
    priority: 10
    state_id: idle
    when: {op: eq, path: x, value: 1}
`,
			wantSub: `expected "guard"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "reinguard.yaml"), []byte(`schema_version: "0.3.0"
default_branch: main
providers: []
`))
			kindDir := filepath.Join(dir, "control", tt.kind)
			if err := os.MkdirAll(kindDir, 0o755); err != nil {
				t.Fatal(err)
			}
			writeFile(t, filepath.Join(kindDir, "bad.yaml"), []byte(tt.yamlBody))

			_, err := Load(dir)
			if err == nil || !strings.Contains(err.Error(), tt.wantSub) {
				t.Fatalf("got err=%v, want substring %q", err, tt.wantSub)
			}
		})
	}
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
