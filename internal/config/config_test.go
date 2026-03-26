package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/k-shibuki/reinguard/pkg/schema"
)

func TestLoad_emptyConfigDir(t *testing.T) {
	t.Parallel()
	// Given: empty config directory path
	// When: Load is called
	_, err := Load("")
	// Then: error mentions empty directory
	if err == nil || !strings.Contains(err.Error(), "empty directory") {
		t.Fatalf("got %v", err)
	}
}

func TestDeprecatedWarnings_nilRoot(t *testing.T) {
	t.Parallel()
	// Given: nil Root
	// When: DeprecatedWarnings is called
	// Then: no warnings
	if w := DeprecatedWarnings(nil); len(w) != 0 {
		t.Fatalf("got %v", w)
	}
}

func TestDeprecatedWarnings_noLegacyHints(t *testing.T) {
	t.Parallel()
	// Given: root without legacy_tool_hints and schema_version matches rgd contract
	r := &Root{SchemaVersion: schema.CurrentSchemaVersion, DefaultBranch: "main"}
	// When: DeprecatedWarnings is called
	// Then: no warnings
	if w := DeprecatedWarnings(r); len(w) != 0 {
		t.Fatalf("got %v", w)
	}
}

func TestDeprecatedWarnings_legacyHints(t *testing.T) {
	t.Parallel()
	// Given: root with legacy_tool_hints and schema_version matches contract (no skew)
	r := &Root{
		SchemaVersion:   schema.CurrentSchemaVersion,
		LegacyToolHints: map[string]any{"x": 1},
		DefaultBranch:   "main",
	}
	// When: DeprecatedWarnings runs
	w := DeprecatedWarnings(r)
	// Then: exactly one warning references legacy_tool_hints
	if len(w) != 1 || !strings.Contains(w[0], "legacy_tool_hints") {
		t.Fatalf("got %v", w)
	}
}

func TestDeprecatedWarnings_schemaSkewAndLegacy(t *testing.T) {
	t.Parallel()
	cm, ci, cp, err := parseSemverCore(schema.CurrentSchemaVersion)
	if err != nil {
		t.Fatal(err)
	}
	if cp == 0 && ci == 0 {
		t.Skip("need non-zero minor or patch on contract to build older skew")
	}
	// Given: older same-major schema_version and legacy_tool_hints
	older := fmt.Sprintf("%d.%d.%d", cm, ci, cp-1)
	if cp == 0 {
		older = fmt.Sprintf("%d.%d.%d", cm, ci-1, 0)
	}
	r := &Root{
		SchemaVersion:   older,
		DefaultBranch:   "main",
		LegacyToolHints: map[string]any{"x": 1},
	}
	// When: DeprecatedWarnings runs
	w := DeprecatedWarnings(r)
	// Then: skew warning first, legacy second
	if len(w) != 2 {
		t.Fatalf("want 2 warnings, got %v", w)
	}
	if !strings.Contains(w[0], "schema_version") {
		t.Fatalf("first line should be schema skew: %q", w[0])
	}
	if !strings.Contains(w[1], "legacy_tool_hints") {
		t.Fatalf("second line should be legacy: %q", w[1])
	}
}

func TestLoad_schemaVersion_policy(t *testing.T) {
	t.Parallel()
	cur := schema.CurrentSchemaVersion
	cm, ci, cp, err := parseSemverCore(cur)
	if err != nil {
		t.Fatal(err)
	}

	var olderSameMajor string
	switch {
	case cp > 0:
		olderSameMajor = fmt.Sprintf("%d.%d.%d", cm, ci, cp-1)
	case ci > 0:
		olderSameMajor = fmt.Sprintf("%d.%d.%d", cm, ci-1, 0)
	default:
		t.Skip("contract at x.0.0 with patch 0 — cannot derive older same-major")
	}
	newerPatch := fmt.Sprintf("%d.%d.%d", cm, ci, cp+1)
	newerMinor := fmt.Sprintf("%d.%d.%d", cm, ci+1, 0)
	majorMismatch := fmt.Sprintf("%d.%d.%d", cm+1, ci, cp)

	tests := []struct {
		name           string
		declared       string
		wantErrSubstr  string
		wantSkewSubstr string
		wantLoadErr    bool
	}{
		{
			name:     "same_as_contract",
			declared: cur,
			// Given: schema_version equals rgd contract
			// When: Load runs
			// Then: success and no skew warning from DeprecatedWarnings
		},
		{
			name:           "older_same_major",
			declared:       olderSameMajor,
			wantSkewSubstr: "schema_version",
			// Given: same major, older minor or patch than contract
			// When: Load runs
			// Then: success; DeprecatedWarnings mentions schema_version skew
		},
		{
			name:           "newer_patch_same_major",
			declared:       newerPatch,
			wantSkewSubstr: "schema_version",
			// Given: same major, newer patch than contract
			// When: Load runs
			// Then: success with skew warning
		},
		{
			name:           "newer_minor_same_major",
			declared:       newerMinor,
			wantSkewSubstr: "schema_version",
			// Given: same major, newer minor than contract
			// When: Load runs
			// Then: success with skew warning
		},
		{
			name:          "major_mismatch",
			declared:      majorMismatch,
			wantLoadErr:   true,
			wantErrSubstr: "incompatible",
			// Given: declared major differs from rgd contract
			// When: Load runs
			// Then: error (no silent load)
		},
		{
			name:     "prerelease_same_core_as_contract",
			declared: cur + "-rc.1",
			// Given: prerelease tag on same MAJOR.MINOR.PATCH as contract
			// When: Load runs
			// Then: success; core matches so no skew warning
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "reinguard.yaml"), []byte(fmt.Sprintf(`schema_version: %q
default_branch: main
providers: []
`, tt.declared)))

			res, lerr := Load(dir)
			if tt.wantLoadErr {
				// Then: Load returns an error
				if lerr == nil || !strings.Contains(lerr.Error(), tt.wantErrSubstr) {
					t.Fatalf("Load: got %v, want err containing %q", lerr, tt.wantErrSubstr)
				}
				return
			}
			// Then: Load succeeds
			if lerr != nil {
				t.Fatal(lerr)
			}
			w := DeprecatedWarnings(&res.Root)
			if tt.wantSkewSubstr != "" {
				if len(w) != 1 || !strings.Contains(w[0], tt.wantSkewSubstr) {
					t.Fatalf("want one skew warning containing %q, got %v", tt.wantSkewSubstr, w)
				}
				return
			}
			if len(w) != 0 {
				t.Fatalf("want no warnings, got %v", w)
			}
		})
	}
}

// reinguardYAMLMinimal returns reinguard.yaml body with schema_version aligned to the binary contract.
func reinguardYAMLMinimal() []byte {
	return []byte(fmt.Sprintf(`schema_version: %q
default_branch: main
providers: []
`, schema.CurrentSchemaVersion))
}

func TestLoad_knowledgeManifest_ok(t *testing.T) {
	t.Parallel()
	// Given: valid root YAML and valid knowledge manifest.json
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), reinguardYAMLMinimal())
	if err := os.Mkdir(filepath.Join(dir, "knowledge"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "knowledge", "manifest.json"), []byte(fmt.Sprintf(`{
  "schema_version": %q,
  "entries": [{
    "id": "doc1",
    "path": "docs/a.md",
    "description": "test doc",
    "triggers": ["test"]
  }]
}`, schema.CurrentSchemaVersion)))

	// When: Load runs
	res, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !res.KnowledgePresent || res.Knowledge == nil {
		t.Fatal("expected knowledge manifest")
	}
	// Then: knowledge manifest is loaded with expected entry
	if len(res.Knowledge.Entries) != 1 || res.Knowledge.Entries[0].ID != "doc1" {
		t.Fatalf("%+v", res.Knowledge)
	}
}

func TestLoad_knowledgeManifest_invalidJSON(t *testing.T) {
	t.Parallel()
	// Given: root OK but manifest.json is invalid JSON
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), reinguardYAMLMinimal())
	if err := os.Mkdir(filepath.Join(dir, "knowledge"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "knowledge", "manifest.json"), []byte(`{`))

	// When: Load runs
	_, err := Load(dir)
	// Then: parse knowledge manifest error
	if err == nil || !strings.Contains(err.Error(), "parse knowledge manifest") {
		t.Fatalf("got %v", err)
	}
}

func TestLoad_knowledgeManifest_schemaInvalid(t *testing.T) {
	t.Parallel()
	// Given: manifest missing required fields
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), reinguardYAMLMinimal())
	if err := os.Mkdir(filepath.Join(dir, "knowledge"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Missing required "entries"
	writeFile(t, filepath.Join(dir, "knowledge", "manifest.json"), []byte(`{"schema_version":"0.1.0"}`))

	// When: Load runs
	_, err := Load(dir)
	// Then: schema validation error
	if err == nil || !strings.Contains(err.Error(), "schema validation") {
		t.Fatalf("got %v", err)
	}
}

func TestLoad_controlStatesDotYmlAndStableOrder(t *testing.T) {
	t.Parallel()
	// Given: two state rule files (.yml and .yaml) under control/states
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), reinguardYAMLMinimal())
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

	// When: Load runs
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
	// Then: both files load and rules sort by id (a before z)
	if len(rules) != 2 || rules[0].ID != "a" || rules[1].ID != "z" {
		t.Fatalf("expected [a, z] ordering, got %+v", rules)
	}
}

func TestLoad_legacyRulesYAML_rejected(t *testing.T) {
	t.Parallel()
	// Given: legacy top-level rules/ directory exists
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), reinguardYAMLMinimal())
	legacyDir := filepath.Join(dir, "rules")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(legacyDir, "old.yaml"), []byte("rules: []\n"))

	// When: Load runs
	_, err := Load(dir)
	// Then: legacy rules rejection error
	if err == nil || !strings.Contains(err.Error(), "legacy rules") {
		t.Fatalf("got %v", err)
	}
}

func TestLoad_minimalValid(t *testing.T) {
	t.Parallel()
	// Given: valid reinguard.yaml and no control/ directory
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), []byte(fmt.Sprintf(`schema_version: %q
default_branch: main
providers:
  - id: git
    enabled: true
`, schema.CurrentSchemaVersion)))

	// When: Load is called
	res, err := Load(dir)

	// Then: No error and schema version matches
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if res.Root.SchemaVersion != schema.CurrentSchemaVersion {
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
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), []byte(fmt.Sprintf(`schema_version: %q
providers: []
`, schema.CurrentSchemaVersion)))

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
	// Given: two providers with the same id
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), []byte(fmt.Sprintf(`schema_version: %q
default_branch: main
providers:
  - id: git
    enabled: true
  - id: git
    enabled: false
`, schema.CurrentSchemaVersion)))

	// When: Load runs
	_, err := Load(dir)
	// Then: duplicate provider id error
	if err == nil || !strings.Contains(err.Error(), "duplicate provider id") {
		t.Fatalf("got err=%v", err)
	}
}

func TestLoad_controlStatesFile(t *testing.T) {
	t.Parallel()
	// Given: valid root and one state rules file
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), reinguardYAMLMinimal())
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
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), reinguardYAMLMinimal())
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
	// Given/When/Then: each case places a rule of the wrong type under control/{states,routes,guards}; Load must error with wantSub
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
			// Given: rule type in wrong control/ subdirectory (see tt)
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "reinguard.yaml"), reinguardYAMLMinimal())
			kindDir := filepath.Join(dir, "control", tt.kind)
			if err := os.MkdirAll(kindDir, 0o755); err != nil {
				t.Fatal(err)
			}
			writeFile(t, filepath.Join(kindDir, "bad.yaml"), []byte(tt.yamlBody))

			// When: Load runs
			_, err := Load(dir)
			// Then: error contains expected mismatch substring
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
