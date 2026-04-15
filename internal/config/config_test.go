package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/k-shibuki/reinguard/pkg/schema"
	"gopkg.in/yaml.v3"
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
	// Given: declared schema_version values vs rgd contract
	// When: Load and DeprecatedWarnings run per row
	// Then: load error, skew warning, or clean success as expected
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
		skipWhen       bool
	}{
		{
			name:     "same_as_contract",
			declared: cur,
		},
		{
			name:           "older_same_major",
			declared:       olderSameMajor,
			skipWhen:       olderSameMajor == "",
			wantSkewSubstr: "schema_version",
		},
		{
			name:           "newer_patch_same_major",
			declared:       newerPatch,
			wantSkewSubstr: "schema_version",
		},
		{
			name:           "newer_minor_same_major",
			declared:       newerMinor,
			wantSkewSubstr: "schema_version",
		},
		{
			name:          "major_mismatch",
			declared:      majorMismatch,
			wantLoadErr:   true,
			wantErrSubstr: "incompatible",
		},
		{
			name:     "prerelease_same_core_as_contract",
			declared: cur + "-rc.1",
		},
		{
			name:     "build_metadata_same_core_as_contract",
			declared: cur + "+build.1",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if tt.skipWhen {
				t.Skip("precondition unmet for this contract version")
			}
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "reinguard.yaml"), []byte(fmt.Sprintf(`schema_version: %q
default_branch: main
providers: []
`, tt.declared)))

			res, lerr := Load(dir)
			if tt.wantLoadErr {
				if lerr == nil || !strings.Contains(lerr.Error(), tt.wantErrSubstr) {
					t.Fatalf("Load: got %v, want err containing %q", lerr, tt.wantErrSubstr)
				}
				return
			}
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
    "triggers": ["test"],
    "when": {"eval": "constant", "params": {"value": true}}
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

func TestLoad_knowledgeManifest_whenUnknownOp(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), reinguardYAMLMinimal())
	if err := os.Mkdir(filepath.Join(dir, "knowledge"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "knowledge", "manifest.json"), []byte(fmt.Sprintf(`{
  "schema_version": %q,
  "entries": [{
    "id": "bad",
    "path": "docs/a.md",
    "description": "d",
    "triggers": ["t"],
    "when": {"op": "bogus", "path": "git.branch", "value": 1}
  }]
}`, schema.CurrentSchemaVersion)))

	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "unknown op") {
		t.Fatalf("got %v", err)
	}
}

func TestLoad_knowledgeManifest_whenPathBadPrefix(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), reinguardYAMLMinimal())
	if err := os.Mkdir(filepath.Join(dir, "knowledge"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "knowledge", "manifest.json"), []byte(fmt.Sprintf(`{
  "schema_version": %q,
  "entries": [{
    "id": "bad",
    "path": "docs/a.md",
    "description": "d",
    "triggers": ["t"],
    "when": {"op": "eq", "path": "invalid.prefix", "value": 1}
  }]
}`, schema.CurrentSchemaVersion)))

	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "must start with git.") {
		t.Fatalf("got %v", err)
	}
}

func TestLoad_knowledgeManifest_whenUnknownEvaluator(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), reinguardYAMLMinimal())
	if err := os.Mkdir(filepath.Join(dir, "knowledge"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "knowledge", "manifest.json"), []byte(fmt.Sprintf(`{
  "schema_version": %q,
  "entries": [{
    "id": "bad",
    "path": "docs/a.md",
    "description": "d",
    "triggers": ["t"],
    "when": {"eval": "nonexistent-evaluator"}
  }]
}`, schema.CurrentSchemaVersion)))

	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "unknown evaluator") {
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
	// Missing required "entries" (schema_version matches contract so JSON Schema runs)
	writeFile(t, filepath.Join(dir, "knowledge", "manifest.json"), []byte(fmt.Sprintf(`{"schema_version":%q}`, schema.CurrentSchemaVersion)))

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
	writeFile(t, filepath.Join(statesDir, "z.yml"), []byte(fmt.Sprintf(`schema_version: %q
rules:
  - type: state
    id: z
    priority: 10
    when:
      op: eq
      path: git.branch
      value: main
    state_id: Z
`, schema.CurrentSchemaVersion)))
	writeFile(t, filepath.Join(statesDir, "a.yaml"), []byte(fmt.Sprintf(`schema_version: %q
rules:
  - type: state
    id: a
    priority: 10
    when:
      op: eq
      path: git.branch
      value: main
    state_id: A
`, schema.CurrentSchemaVersion)))
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

func TestLoad_withLabelsYaml_ok(t *testing.T) {
	t.Parallel()
	// Given: valid reinguard.yaml and minimal labels.yaml next to it
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), []byte(fmt.Sprintf(`schema_version: %q
default_branch: main
providers:
  - id: git
    enabled: true
`, schema.CurrentSchemaVersion)))
	writeFile(t, filepath.Join(dir, "labels.yaml"), []byte(fmt.Sprintf(`schema_version: %q
categories:
  type:
    description: "Conventional Commits type labels"
    scope: shared
    labels:
      - name: feat
        color: "A2EEEF"
        description: "Type: new feature"
`, schema.CurrentSchemaVersion)))

	// When: Load is called
	res, err := Load(dir)

	// Then: labels are parsed and validated
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !res.LabelsPresent || res.Labels == nil {
		t.Fatal("expected LabelsPresent and non-nil Labels")
	}
	if res.Labels.SchemaVersion != schema.CurrentSchemaVersion {
		t.Fatalf("labels schema_version: got %q", res.Labels.SchemaVersion)
	}
	if got := res.Labels.TypeLabelNames(); len(got) != 1 || got[0] != "feat" {
		t.Fatalf("TypeLabelNames: %v", got)
	}
}

func TestLoad_labelsYaml_schemaInvalid(t *testing.T) {
	t.Parallel()
	// Given: valid reinguard.yaml and labels.yaml that fails JSON Schema (empty categories)
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), []byte(fmt.Sprintf(`schema_version: %q
default_branch: main
providers:
  - id: git
    enabled: true
`, schema.CurrentSchemaVersion)))
	writeFile(t, filepath.Join(dir, "labels.yaml"), []byte(fmt.Sprintf(`schema_version: %q
categories: {}
`, schema.CurrentSchemaVersion)))

	// When: Load is called
	_, err := Load(dir)

	// Then: schema validation error for labels.yaml
	if err == nil || !strings.Contains(err.Error(), "labels.yaml") || !strings.Contains(err.Error(), "schema validation") {
		t.Fatalf("expected schema validation error for labels.yaml, got %v", err)
	}
}

func TestLoad_labelsYaml_parseError(t *testing.T) {
	t.Parallel()
	// Given: labels.yaml that is not valid YAML
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), []byte(fmt.Sprintf(`schema_version: %q
default_branch: main
providers:
  - id: git
    enabled: true
`, schema.CurrentSchemaVersion)))
	writeFile(t, filepath.Join(dir, "labels.yaml"), []byte("categories: [\n"))

	// When: Load is called
	_, err := Load(dir)

	// Then: parse error references labels.yaml
	if err == nil || !strings.Contains(err.Error(), "labels.yaml") || !strings.Contains(err.Error(), "parse") {
		t.Fatalf("expected parse error for labels.yaml, got %v", err)
	}
}

func TestLoad_labelsYaml_incompatibleMajor(t *testing.T) {
	t.Parallel()
	// Given: labels.yaml with schema_version major incompatible with rgd
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), []byte(fmt.Sprintf(`schema_version: %q
default_branch: main
providers:
  - id: git
    enabled: true
`, schema.CurrentSchemaVersion)))
	writeFile(t, filepath.Join(dir, "labels.yaml"), []byte(`schema_version: "99.0.0"
categories:
  type:
    description: "t"
    scope: shared
    labels:
      - name: feat
        color: "A2EEEF"
        description: "x"
`))

	// When: Load is called
	_, err := Load(dir)

	// Then: error from declared schema version check
	if err == nil || !strings.Contains(err.Error(), "incompatible") {
		t.Fatalf("expected incompatible schema_version error, got %v", err)
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

func TestLoadRoot_runtimeGateRolesOverride(t *testing.T) {
	t.Parallel()
	// Given: reinguard.yaml with workflow.runtime_gate_roles override for pre_pr_ai_review
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), []byte(fmt.Sprintf(`schema_version: %q
default_branch: main
workflow:
  runtime_gate_roles:
    pre_pr_ai_review:
      gate_id: local-ai-review
      required: false
      producer_procedures: [change-inspect]
      pass_check_ids: [ai-review-cli]
providers: []
`, schema.CurrentSchemaVersion)))

	// When: LoadRoot is called
	root, err := LoadRoot(dir)
	// Then: override applies and unrelated defaults remain intact
	if err != nil {
		t.Fatal(err)
	}
	roles := root.EffectiveRuntimeGateRoles()
	if roles.PrePRAIReview.GateID != "local-ai-review" {
		t.Fatalf("pre_pr_ai_review gate_id=%q", roles.PrePRAIReview.GateID)
	}
	if roles.PrePRAIReview.Required == nil || *roles.PrePRAIReview.Required {
		t.Fatalf("pre_pr_ai_review required=%v", roles.PrePRAIReview.Required)
	}
	if roles.PRReadiness.GateID != "pr-readiness" {
		t.Fatalf("default pr_readiness should remain intact: %+v", roles.PRReadiness)
	}
}

func TestLoadRoot_runtimeGateRolesDuplicateGateID(t *testing.T) {
	t.Parallel()
	// Given: reinguard.yaml with two runtime gate roles sharing the same gate_id
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), []byte(fmt.Sprintf(`schema_version: %q
default_branch: main
workflow:
  runtime_gate_roles:
    local_verification:
      gate_id: shared-gate
    pr_readiness:
      gate_id: shared-gate
providers: []
`, schema.CurrentSchemaVersion)))

	// When: LoadRoot is called
	_, err := LoadRoot(dir)
	// Then: error containing "duplicates" is returned
	if err == nil || !strings.Contains(err.Error(), "duplicates") {
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
	writeFile(t, rulePath, []byte(fmt.Sprintf(`schema_version: %q
rules:
  - type: state
    id: s1
    priority: 10
    when:
      op: eq
      path: git.branch
      value: main
    state_id: Idle
`, schema.CurrentSchemaVersion)))

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

func TestLoad_evaluatorReferencesInWhen_table(t *testing.T) {
	t.Parallel()
	// Given: minimal config plus guard rules with varied when shapes
	// When: Load runs
	// Then: success or unknown-evaluator error per row
	tests := []struct {
		name       string
		whenYAML   string
		wantErrSub string
	}{
		{
			name: "valid_constant_top_level",
			whenYAML: `
      eval: constant
      params:
        value: true`,
		},
		{
			name: "valid_constant_nested_in_and",
			whenYAML: `
      and:
        - op: eq
          path: git.branch
          value: main
        - eval: constant
          params:
            value: true`,
		},
		{
			name: "valid_constant_nested_in_count_when",
			whenYAML: `
      op: count
      path: state.items
      eq: 0
      when:
        eval: constant
        params:
          value: true`,
		},
		{
			name: "unknown_evaluator_name",
			whenYAML: `
      eval: no-such-evaluator`,
			wantErrSub: "unknown evaluator",
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "reinguard.yaml"), reinguardYAMLMinimal())
			guardsDir := filepath.Join(dir, "control", "guards")
			if err := os.MkdirAll(guardsDir, 0o755); err != nil {
				t.Fatal(err)
			}
			body := fmt.Sprintf(`schema_version: %q
rules:
  - type: guard
    id: g1
    priority: 1
    guard_id: merge-readiness
    when:`+tc.whenYAML+"\n", schema.CurrentSchemaVersion)
			writeFile(t, filepath.Join(guardsDir, "g.yaml"), []byte(body))

			res, err := Load(dir)
			if tc.wantErrSub != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErrSub) {
					t.Fatalf("Load() err=%v want substring %q", err, tc.wantErrSub)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			rs := res.Rules()
			if len(rs) != 1 || rs[0].ID != "g1" {
				t.Fatalf("rules: %+v", rs)
			}
		})
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
	writeFile(t, filepath.Join(statesDir, "bad.yaml"), []byte(fmt.Sprintf(`schema_version: %q
rules: not-an-array
`, schema.CurrentSchemaVersion)))

	// When: Load is called
	_, err := Load(dir)

	// Then: validation error names rules file
	if err == nil || !strings.Contains(err.Error(), "bad.yaml") || !strings.Contains(err.Error(), "schema validation") {
		t.Fatalf("%v", err)
	}
}

func TestLoad_controlKindTypeMismatchRejected(t *testing.T) {
	t.Parallel()
	// Given: control rules whose type does not match their control/ subdirectory
	// When: Load runs
	// Then: error contains expected type mismatch substring
	tests := []struct {
		name     string
		kind     string
		yamlBody string
		wantSub  string
	}{
		{
			name: "states_dir_has_route_type",
			kind: "states",
			yamlBody: fmt.Sprintf(`schema_version: %q
rules:
  - type: route
    id: r1
    priority: 10
    route_id: next
    when: {op: eq, path: x, value: 1}
`, schema.CurrentSchemaVersion),
			wantSub: `expected "state"`,
		},
		{
			name: "routes_dir_has_state_type",
			kind: "routes",
			yamlBody: fmt.Sprintf(`schema_version: %q
rules:
  - type: state
    id: s1
    priority: 10
    state_id: idle
    when: {op: eq, path: x, value: 1}
`, schema.CurrentSchemaVersion),
			wantSub: `expected "route"`,
		},
		{
			name: "guards_dir_has_state_type",
			kind: "guards",
			yamlBody: fmt.Sprintf(`schema_version: %q
rules:
  - type: state
    id: s1
    priority: 10
    state_id: idle
    when: {op: eq, path: x, value: 1}
`, schema.CurrentSchemaVersion),
			wantSub: `expected "guard"`,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			writeFile(t, filepath.Join(dir, "reinguard.yaml"), reinguardYAMLMinimal())
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

func TestSchemaVersionSync_committedVersionedFiles(t *testing.T) {
	t.Parallel()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	relPaths := []string{
		".reinguard/reinguard.yaml",
		".reinguard/labels.yaml",
		"internal/labels/data/labels.yaml",
		".reinguard/knowledge/manifest.json",
		".reinguard/policy/catalog.yaml",
		".reinguard/control/catalog.yaml",
		".reinguard/control/states/workflow.yaml",
		".reinguard/control/routes/workflow.yaml",
		".reinguard/control/guards/default.yaml",
	}
	for _, rel := range relPaths {
		rel := rel
		t.Run(rel, func(t *testing.T) {
			t.Parallel()
			path := filepath.Join(root, rel)
			data, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("read %s: %v", path, err)
			}
			var got string
			switch filepath.Ext(rel) {
			case ".json":
				var m map[string]any
				if err := json.Unmarshal(data, &m); err != nil {
					t.Fatalf("json %s: %v", rel, err)
				}
				sv, ok := m["schema_version"].(string)
				if !ok || strings.TrimSpace(sv) == "" {
					t.Fatalf("%s: missing string schema_version", rel)
				}
				got = sv
			default:
				var m map[string]any
				if err := yaml.Unmarshal(data, &m); err != nil {
					t.Fatalf("yaml %s: %v", rel, err)
				}
				sv, ok := m["schema_version"].(string)
				if !ok || strings.TrimSpace(sv) == "" {
					t.Fatalf("%s: missing string schema_version", rel)
				}
				got = sv
			}
			if got != schema.CurrentSchemaVersion {
				t.Fatalf("%s: schema_version %q, want %q", rel, got, schema.CurrentSchemaVersion)
			}
		})
	}
}

func olderSameMajorBelowContract(t *testing.T) string {
	t.Helper()
	cur := schema.CurrentSchemaVersion
	cm, ci, cp, err := parseSemverCore(cur)
	if err != nil {
		t.Fatal(err)
	}
	if cp > 0 {
		return fmt.Sprintf("%d.%d.%d", cm, ci, cp-1)
	}
	if ci > 0 {
		return fmt.Sprintf("%d.%d.%d", cm, ci-1, 0)
	}
	t.Skip("need older same-major skew fixture")
	return ""
}

func TestConfigWarnings_labelsSkew(t *testing.T) {
	t.Parallel()
	older := olderSameMajorBelowContract(t)
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), reinguardYAMLMinimal())
	writeFile(t, filepath.Join(dir, "labels.yaml"), []byte(fmt.Sprintf(`schema_version: %q
categories:
  x:
    description: d
    scope: issue
    labels:
      - name: a
        color: "000000"
        description: d
`, older)))

	res, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	w := ConfigWarnings(res)
	if len(w) != 1 || !strings.Contains(w[0], "labels.yaml") || !strings.Contains(w[0], "schema_version") {
		t.Fatalf("want one labels skew warning, got %v", w)
	}
}

func TestConfigWarnings_manifestSkew(t *testing.T) {
	t.Parallel()
	older := olderSameMajorBelowContract(t)
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), reinguardYAMLMinimal())
	if err := os.Mkdir(filepath.Join(dir, "knowledge"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "knowledge", "manifest.json"), []byte(fmt.Sprintf(`{
  "schema_version": %q,
  "entries": []
}`, older)))

	res, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	w := ConfigWarnings(res)
	if len(w) != 1 || !strings.Contains(w[0], "manifest.json") || !strings.Contains(w[0], "schema_version") {
		t.Fatalf("want one manifest skew warning, got %v", w)
	}
}

func TestConfigWarnings_controlStateSkew(t *testing.T) {
	t.Parallel()
	older := olderSameMajorBelowContract(t)
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), reinguardYAMLMinimal())
	statesDir := filepath.Join(dir, "control", "states")
	if err := os.MkdirAll(statesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(statesDir, "x.yaml"), []byte(fmt.Sprintf(`schema_version: %q
rules:
  - type: state
    id: s1
    priority: 10
    state_id: idle
    when: {op: eq, path: git.branch, value: main}
`, older)))

	res, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	w := ConfigWarnings(res)
	if len(w) != 1 || !strings.Contains(w[0], "control/states/x.yaml") || !strings.Contains(w[0], "schema_version") {
		t.Fatalf("want one control skew warning, got %v", w)
	}
}

func TestLoad_controlRule_schemaMajorMismatch(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), reinguardYAMLMinimal())
	statesDir := filepath.Join(dir, "control", "states")
	if err := os.MkdirAll(statesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(statesDir, "bad.yaml"), []byte(`schema_version: "99.0.0"
rules: []
`))

	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "incompatible") {
		t.Fatalf("got %v", err)
	}
}

func TestLoad_procedureMapping_ok(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), reinguardYAMLMinimal())
	statesDir := filepath.Join(dir, "control", "states")
	if err := os.MkdirAll(statesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(statesDir, "t.yaml"), []byte(fmt.Sprintf(`schema_version: %q
rules:
  - type: state
    id: only
    priority: 10
    when:
      op: eq
      path: git.branch
      value: main
    state_id: working_no_pr
`, schema.CurrentSchemaVersion)))
	procDir := filepath.Join(dir, "procedure")
	if err := os.MkdirAll(procDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(procDir, "p.md"), []byte(`---
id: procedure-test
purpose: Test procedure.
applies_to:
  state_ids:
    - working_no_pr
  route_ids: []
---
`))
	res, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !res.ProcedurePresent || len(res.ProcedureEntries) != 1 {
		t.Fatalf("procedure: present=%v entries=%+v", res.ProcedurePresent, res.ProcedureEntries)
	}
}

func TestLoad_procedureMapping_duplicateStateRejected(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), reinguardYAMLMinimal())
	statesDir := filepath.Join(dir, "control", "states")
	if err := os.MkdirAll(statesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(statesDir, "t.yaml"), []byte(fmt.Sprintf(`schema_version: %q
rules:
  - type: state
    id: only
    priority: 10
    when:
      op: eq
      path: git.branch
      value: main
    state_id: working_no_pr
`, schema.CurrentSchemaVersion)))
	procDir := filepath.Join(dir, "procedure")
	if err := os.MkdirAll(procDir, 0o755); err != nil {
		t.Fatal(err)
	}
	body := `---
id: procedure-a
purpose: A
applies_to:
  state_ids:
    - working_no_pr
  route_ids: []
---
`
	writeFile(t, filepath.Join(procDir, "a.md"), []byte(body))
	writeFile(t, filepath.Join(procDir, "b.md"), []byte(strings.Replace(body, "procedure-a", "procedure-b", 1)))

	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "state_id \"working_no_pr\"") {
		t.Fatalf("got %v", err)
	}
}

func TestLoad_procedureMapping_orphanStateRejected(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), reinguardYAMLMinimal())
	statesDir := filepath.Join(dir, "control", "states")
	if err := os.MkdirAll(statesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(statesDir, "t.yaml"), []byte(fmt.Sprintf(`schema_version: %q
rules:
  - type: state
    id: only
    priority: 10
    when:
      op: eq
      path: git.branch
      value: main
    state_id: working_no_pr
`, schema.CurrentSchemaVersion)))
	procDir := filepath.Join(dir, "procedure")
	if err := os.MkdirAll(procDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(procDir, "p.md"), []byte(`---
id: procedure-bad
purpose: Bad mapping.
applies_to:
  state_ids:
    - not_a_real_state
  route_ids: []
---
`))

	_, err := Load(dir)
	if err == nil || !strings.Contains(err.Error(), "no matching control state rule") {
		t.Fatalf("got %v", err)
	}
}

func TestLoad_procedureDirAbsent_ok(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), reinguardYAMLMinimal())
	statesDir := filepath.Join(dir, "control", "states")
	if err := os.MkdirAll(statesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(statesDir, "t.yaml"), []byte(fmt.Sprintf(`schema_version: %q
rules:
  - type: state
    id: only
    priority: 10
    when:
      op: eq
      path: git.branch
      value: main
    state_id: working_no_pr
`, schema.CurrentSchemaVersion)))
	res, err := Load(dir)
	if err != nil {
		t.Fatal(err)
	}
	if res.ProcedurePresent || len(res.ProcedureEntries) != 0 {
		t.Fatalf("expected no procedure bundle, got present=%v n=%d", res.ProcedurePresent, len(res.ProcedureEntries))
	}
}

func TestLoad_repositoryReinguard(t *testing.T) {
	t.Parallel()
	// Given: this repository's committed .reinguard bundle (control + knowledge)
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	dir := filepath.Join(root, ".reinguard")
	if _, err := os.Stat(dir); err != nil {
		t.Fatalf("stat %s: %v", dir, err)
	}
	// When: Load runs
	// Then: no error (schema + when validation for FSM YAML)
	res, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if !res.ProcedurePresent {
		t.Fatal("expected .reinguard/procedure to be loaded")
	}
	if len(res.ProcedureEntries) < 1 {
		t.Fatalf("expected procedure entries, got %d", len(res.ProcedureEntries))
	}

	// And: control catalog lists workflow bundle entries expected by adapter/docs
	assertReinguardControlCatalogWorkflowEntries(t, dir)
}

// assertReinguardControlCatalogWorkflowEntries checks control/catalog.yaml maps the
// FSM workflow bundle with id/path/type/description per entry and files on disk.
func assertReinguardControlCatalogWorkflowEntries(t *testing.T, reinguardDir string) {
	t.Helper()
	catalogPath := filepath.Join(reinguardDir, "control", "catalog.yaml")
	data, err := os.ReadFile(catalogPath)
	if err != nil {
		t.Fatalf("read %s: %v", catalogPath, err)
	}
	var cat struct {
		Entries []struct {
			ID          string `yaml:"id"`
			Path        string `yaml:"path"`
			Type        string `yaml:"type"`
			Description string `yaml:"description"`
		} `yaml:"entries"`
	}
	if err := yaml.Unmarshal(data, &cat); err != nil {
		t.Fatalf("catalog yaml: %v", err)
	}
	byID := make(map[string]struct {
		Path        string
		Type        string
		Description string
	})
	for _, e := range cat.Entries {
		if strings.TrimSpace(e.ID) == "" {
			t.Fatal("catalog has blank entry id")
		}
		if _, exists := byID[e.ID]; exists {
			t.Fatalf("catalog has duplicate entry id %q", e.ID)
		}
		byID[e.ID] = struct {
			Path        string
			Type        string
			Description string
		}{e.Path, e.Type, e.Description}
	}
	for _, w := range []struct {
		id   string
		path string
		typ  string
	}{
		{"workflow-states", "states/workflow.yaml", "state"},
		{"workflow-routes", "routes/workflow.yaml", "route"},
		{"merge-readiness-guard", "guards/default.yaml", "guard"},
	} {
		e, ok := byID[w.id]
		if !ok {
			t.Fatalf("catalog missing entry id %q", w.id)
		}
		if e.Path != w.path {
			t.Fatalf("catalog %q path: got %q want %q", w.id, e.Path, w.path)
		}
		if e.Type != w.typ {
			t.Fatalf("catalog %q type: got %q want %q", w.id, e.Type, w.typ)
		}
		if strings.TrimSpace(e.Description) == "" {
			t.Fatalf("catalog %q missing non-empty description", w.id)
		}
		target := filepath.Join(reinguardDir, "control", e.Path)
		st, err := os.Stat(target)
		if err != nil {
			t.Fatalf("catalog %q path %q: missing on disk: %v", w.id, e.Path, err)
		}
		if st.IsDir() {
			t.Fatalf("catalog %q path %q: expected file, got directory", w.id, e.Path)
		}
	}
}

func TestEffectiveRuntimeGateRoles_explicitEmptyPassRequiresRoles(t *testing.T) {
	t.Parallel()
	// Given: explicit empty pass_requires_roles for pr_readiness in YAML
	cfg := `schema_version: "0.7.0"
default_branch: main
workflow:
  runtime_gate_roles:
    local_verification:
      gate_id: local-verification
    pre_pr_ai_review:
      gate_id: local-coderabbit
    pr_readiness:
      gate_id: pr-readiness
      pass_requires_roles: []
providers: []
`
	var root Root
	if err := yaml.Unmarshal([]byte(cfg), &root); err != nil {
		t.Fatal(err)
	}
	got := root.EffectiveRuntimeGateRoles().PRReadiness.PassRequiresRoles
	if got == nil || len(*got) != 0 {
		t.Fatalf("want explicit empty pass_requires_roles, got %#v", got)
	}
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
