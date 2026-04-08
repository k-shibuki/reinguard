// Package config loads and validates .reinguard configuration (ADR-0008).
package config

import (
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"

	"github.com/k-shibuki/reinguard/internal/labels"
	"github.com/k-shibuki/reinguard/pkg/schema"
)

// Root is the parsed reinguard.yaml root document.
type Root struct {
	// Defaults is allowed by schema (pkg/schema/reinguard-config.json) for future default rule/provider values; Load does not apply it yet.
	Defaults        map[string]any `yaml:"defaults,omitempty" json:"defaults,omitempty"`
	LegacyToolHints map[string]any `yaml:"legacy_tool_hints,omitempty" json:"legacy_tool_hints,omitempty"`
	SchemaVersion   string         `yaml:"schema_version" json:"schema_version"`
	DefaultBranch   string         `yaml:"default_branch" json:"default_branch"`
	Workflow        WorkflowSpec   `yaml:"workflow,omitempty" json:"workflow,omitempty"`
	Providers       []ProviderSpec `yaml:"providers" json:"providers"`
}

// WorkflowSpec is top-level workflow-specific config that still belongs to repo-owned Semantics.
type WorkflowSpec struct {
	RuntimeGateRoles RuntimeGateRolesSpec `yaml:"runtime_gate_roles,omitempty" json:"runtime_gate_roles,omitempty"`
}

// RuntimeGateRolesSpec declares repo-specific runtime gate role bindings.
type RuntimeGateRolesSpec struct {
	LocalVerification RuntimeGateRoleSpec `yaml:"local_verification,omitempty" json:"local_verification,omitempty"`
	PrePRAIReview     RuntimeGateRoleSpec `yaml:"pre_pr_ai_review,omitempty" json:"pre_pr_ai_review,omitempty"`
	PRReadiness       RuntimeGateRoleSpec `yaml:"pr_readiness,omitempty" json:"pr_readiness,omitempty"`
}

// RuntimeGateRoleSpec binds one semantic gate role to a concrete gate artifact contract.
type RuntimeGateRoleSpec struct {
	Required *bool `yaml:"required,omitempty" json:"required,omitempty"`
	// PassRequiresRoles is optional in YAML: nil means "inherit default"; a non-nil pointer,
	// including to an empty slice, means an explicit override (empty clears upstream roles).
	PassRequiresRoles  *[]string `yaml:"pass_requires_roles,omitempty" json:"pass_requires_roles,omitempty"`
	GateID             string    `yaml:"gate_id,omitempty" json:"gate_id,omitempty"`
	ProducerProcedures []string  `yaml:"producer_procedures,omitempty" json:"producer_procedures,omitempty"`
	PassCheckIDs       []string  `yaml:"pass_check_ids,omitempty" json:"pass_check_ids,omitempty"`
}

// ProviderSpec is one observation provider entry.
type ProviderSpec struct {
	Options map[string]any `yaml:"options,omitempty" json:"options,omitempty"`
	ID      string         `yaml:"id" json:"id"`
	Enabled bool           `yaml:"enabled" json:"enabled"`
}

// Rule is one rule from control/{states,routes,guards}/*.yaml (ADR-0011).
type Rule struct {
	When      any      `yaml:"when" json:"when"`
	Type      string   `yaml:"type" json:"type"`
	ID        string   `yaml:"id" json:"id"`
	StateID   string   `yaml:"state_id,omitempty" json:"state_id,omitempty"`
	RouteID   string   `yaml:"route_id,omitempty" json:"route_id,omitempty"`
	GuardID   string   `yaml:"guard_id,omitempty" json:"guard_id,omitempty"`
	DependsOn []string `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
	Priority  float64  `yaml:"priority" json:"priority"`
}

// RulesDocument wraps the rules array in a YAML file.
type RulesDocument struct {
	SchemaVersion string `yaml:"schema_version" json:"schema_version"`
	Rules         []Rule `yaml:"rules" json:"rules"`
}

// KnowledgeManifestEntry is one knowledge file entry in manifest.json (ADR-0010).
//
//nolint:govet // fieldalignment: keep JSON field order (id, path, description, …) for readable manifests
type KnowledgeManifestEntry struct {
	ID          string   `json:"id"`
	Path        string   `json:"path"`
	Description string   `json:"description"`
	Triggers    []string `json:"triggers"`
	When        any      `json:"when,omitempty"`
}

// KnowledgeManifest is .reinguard/knowledge/manifest.json.
type KnowledgeManifest struct {
	SchemaVersion string                   `json:"schema_version"`
	Entries       []KnowledgeManifestEntry `json:"entries"`
}

// LoadResult holds validated configuration from a config directory.
type LoadResult struct {
	RuleFiles        map[string]RulesDocument
	Knowledge        *KnowledgeManifest
	Labels           *labels.Config
	Dir              string
	Root             Root
	KnowledgePresent bool
	LabelsPresent    bool
}

// Load reads reinguard.yaml, all control/{states,routes,guards}/*.yaml, and optional knowledge/manifest.json.
func Load(dir string) (*LoadResult, error) {
	if dir == "" {
		return nil, fmt.Errorf("config: empty directory")
	}
	ss, err := compileLoadSchemas()
	if err != nil {
		return nil, err
	}
	root, err := readAndValidateRoot(dir, ss.root)
	if err != nil {
		return nil, err
	}
	ruleFiles, err := readControlRuleFiles(dir, ss.rules)
	if err != nil {
		return nil, err
	}
	if err := validateEvaluatorReferences(ruleFiles); err != nil {
		return nil, err
	}
	res := &LoadResult{Dir: dir, Root: root, RuleFiles: ruleFiles}
	if err := applyOptionalLabels(res, dir, ss.labels); err != nil {
		return nil, err
	}
	if err := applyOptionalKnowledge(res, dir, ss.km); err != nil {
		return nil, err
	}
	return res, nil
}

// LoadRoot reads and validates only reinguard.yaml.
func LoadRoot(dir string) (Root, error) {
	if dir == "" {
		return Root{}, fmt.Errorf("config: empty directory")
	}
	ss, err := compileLoadSchemas()
	if err != nil {
		return Root{}, err
	}
	return readAndValidateRoot(dir, ss.root)
}

// Rules returns all rules from all files in stable sort order by control path (kind/filename) then index.
func (r *LoadResult) Rules() []Rule {
	var names []string
	for n := range r.RuleFiles {
		names = append(names, n)
	}
	sort.Strings(names)
	var out []Rule
	for _, n := range names {
		out = append(out, r.RuleFiles[n].Rules...)
	}
	return out
}

func validateRulesMatchControlKind(kind string, rules []Rule, pathHint string) error {
	want := ""
	switch kind {
	case "states":
		want = "state"
	case "routes":
		want = "route"
	case "guards":
		want = "guard"
	default:
		return fmt.Errorf("config: unknown control kind %q", kind)
	}
	for i, ru := range rules {
		if ru.Type != want {
			return fmt.Errorf("config: rule[%d] in %s has type %q, expected %q for control/%s/", i, pathHint, ru.Type, want, kind)
		}
	}
	return nil
}

func validateDoc(sch *jsonschema.Schema, doc any, pathHint string) error {
	if err := sch.Validate(doc); err != nil {
		return fmt.Errorf("config: schema validation %s: %w", pathHint, err)
	}
	return nil
}

func validateUniqueProviderIDs(root *Root, pathHint string) error {
	seen := make(map[string]struct{})
	for _, p := range root.Providers {
		id := strings.TrimSpace(p.ID)
		if id == "" {
			continue
		}
		if _, ok := seen[id]; ok {
			return fmt.Errorf("config: duplicate provider id %q in %s", id, pathHint)
		}
		seen[id] = struct{}{}
	}
	return nil
}

// parseSemverCore extracts MAJOR.MINOR.PATCH integers from a semver string (optional leading "v";
// prerelease / build metadata after the first '-' or '+' is ignored for comparison).
func parseSemverCore(s string) (major, minor, patch int, err error) {
	s = strings.TrimSpace(s)
	if len(s) > 0 && (s[0] == 'v' || s[0] == 'V') {
		s = s[1:]
	}
	if i := strings.IndexByte(s, '+'); i >= 0 {
		s = s[:i]
	}
	if i := strings.IndexByte(s, '-'); i >= 0 {
		s = s[:i]
	}
	parts := strings.Split(s, ".")
	if len(parts) != 3 {
		return 0, 0, 0, fmt.Errorf("expected major.minor.patch, got %q", s)
	}
	major, err = strconv.Atoi(parts[0])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("major: %w", err)
	}
	minor, err = strconv.Atoi(parts[1])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("minor: %w", err)
	}
	patch, err = strconv.Atoi(parts[2])
	if err != nil {
		return 0, 0, 0, fmt.Errorf("patch: %w", err)
	}
	return major, minor, patch, nil
}

func validateDeclaredSchemaVersion(declared, pathHint string) error {
	dm, _, _, derr := parseSemverCore(declared)
	if derr != nil {
		return fmt.Errorf("config: schema_version in %s: %w", pathHint, derr)
	}
	cm, _, _, cerr := parseSemverCore(schema.CurrentSchemaVersion)
	if cerr != nil {
		return fmt.Errorf("config: internal schema contract version: %w", cerr)
	}
	if dm != cm {
		return fmt.Errorf(
			"config: schema_version %q in %s is incompatible with this rgd (major %d vs %d); bump repo or upgrade rgd (ADR-0008, docs/cli.md)",
			declared, pathHint, dm, cm,
		)
	}
	return nil
}

func schemaVersionSkewWarning(declared string) string {
	dm, di, dp, derr := parseSemverCore(declared)
	if derr != nil {
		return ""
	}
	cm, ci, cp, cerr := parseSemverCore(schema.CurrentSchemaVersion)
	if cerr != nil {
		return ""
	}
	if dm != cm {
		return ""
	}
	if di == ci && dp == cp {
		return ""
	}
	return fmt.Sprintf(
		`config warning: schema_version %q differs from this rgd contract %q (same major %d); validation continues — align versions when convenient (see docs/cli.md)`,
		declared, schema.CurrentSchemaVersion, dm,
	)
}

// schemaVersionSkewWarningAt is like schemaVersionSkewWarning but names the declaring file for multi-file skew diagnostics.
func schemaVersionSkewWarningAt(declared, pathHint string) string {
	w := schemaVersionSkewWarning(declared)
	if w == "" {
		return ""
	}
	if pathHint == "" {
		return w
	}
	return fmt.Sprintf("config warning: in %s: %s", pathHint, strings.TrimPrefix(w, "config warning: "))
}

// DeprecatedWarnings returns human-readable stderr lines for deprecated fields and schema skew (ADR-0008).
func DeprecatedWarnings(root *Root) []string {
	if root == nil {
		return nil
	}
	var out []string
	if w := schemaVersionSkewWarning(root.SchemaVersion); w != "" {
		out = append(out, w)
	}
	if len(root.LegacyToolHints) > 0 {
		out = append(out, `config warning: "legacy_tool_hints" is deprecated; remove it from reinguard.yaml (see JSON Schema / docs/cli.md)`)
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// ConfigWarnings aggregates schema skew and deprecation warnings for a full validated Load (reinguard.yaml,
// optional labels and knowledge, and all control rule files). Use after config.Load succeeds.
func ConfigWarnings(res *LoadResult) []string {
	if res == nil {
		return nil
	}
	var out []string
	if w := schemaVersionSkewWarningAt(res.Root.SchemaVersion, "reinguard.yaml"); w != "" {
		out = append(out, w)
	}
	if len(res.Root.LegacyToolHints) > 0 {
		out = append(out, `config warning: "legacy_tool_hints" is deprecated; remove it from reinguard.yaml (see JSON Schema / docs/cli.md)`)
	}
	if res.LabelsPresent && res.Labels != nil {
		if w := schemaVersionSkewWarningAt(res.Labels.SchemaVersion, "labels.yaml"); w != "" {
			out = append(out, w)
		}
	}
	if res.KnowledgePresent && res.Knowledge != nil {
		if w := schemaVersionSkewWarningAt(res.Knowledge.SchemaVersion, filepath.Join("knowledge", "manifest.json")); w != "" {
			out = append(out, w)
		}
	}
	var keys []string
	for k := range res.RuleFiles {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		doc := res.RuleFiles[k]
		hint := filepath.Join("control", filepath.ToSlash(k))
		if w := schemaVersionSkewWarningAt(doc.SchemaVersion, hint); w != "" {
			out = append(out, w)
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// EnabledProviderIDs returns provider ids where enabled is true.
func (r *Root) EnabledProviderIDs() []string {
	var ids []string
	for _, p := range r.Providers {
		if p.Enabled {
			ids = append(ids, strings.TrimSpace(p.ID))
		}
	}
	return ids
}
