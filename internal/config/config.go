// Package config loads and validates .reinguard configuration (ADR-0008).
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	jsonschema "github.com/santhosh-tekuri/jsonschema/v6"
	"gopkg.in/yaml.v3"

	"github.com/k-shibuki/reinguard/pkg/schema"
)

// Root is the parsed reinguard.yaml root document.
type Root struct {
	// Defaults is allowed by schema (pkg/schema/reinguard-config.json) for future default rule/provider values; Load does not apply it yet.
	Defaults        map[string]any `yaml:"defaults,omitempty" json:"defaults,omitempty"`
	LegacyToolHints map[string]any `yaml:"legacy_tool_hints,omitempty" json:"legacy_tool_hints,omitempty"`
	SchemaVersion   string         `yaml:"schema_version" json:"schema_version"`
	DefaultBranch   string         `yaml:"default_branch" json:"default_branch"`
	Providers       []ProviderSpec `yaml:"providers" json:"providers"`
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
	Rules []Rule `yaml:"rules" json:"rules"`
}

// KnowledgeManifestEntry is one knowledge file entry in manifest.json (ADR-0010).
type KnowledgeManifestEntry struct {
	ID          string   `json:"id"`
	Path        string   `json:"path"`
	Description string   `json:"description"`
	Triggers    []string `json:"triggers"`
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
	Dir              string
	Root             Root
	KnowledgePresent bool
}

// Load reads reinguard.yaml, all control/{states,routes,guards}/*.yaml, and optional knowledge/manifest.json.
func Load(dir string) (*LoadResult, error) {
	if dir == "" {
		return nil, fmt.Errorf("config: empty directory")
	}
	comp, err := schema.NewCompiler()
	if err != nil {
		return nil, err
	}
	rootSch, err := comp.Compile(schema.URIReinguardConfig)
	if err != nil {
		return nil, fmt.Errorf("config: compile root schema: %w", err)
	}
	rulesSch, err := comp.Compile(schema.URIRulesDocument)
	if err != nil {
		return nil, fmt.Errorf("config: compile rules schema: %w", err)
	}
	kmSch, err := comp.Compile(schema.URIKnowledgeManifest)
	if err != nil {
		return nil, fmt.Errorf("config: compile knowledge manifest schema: %w", err)
	}

	rootPath := filepath.Join(dir, "reinguard.yaml")
	rootData, err := os.ReadFile(rootPath)
	if err != nil {
		return nil, fmt.Errorf("config: read %s: %w", rootPath, err)
	}
	var rootMap map[string]any
	if err = yaml.Unmarshal(rootData, &rootMap); err != nil {
		return nil, fmt.Errorf("config: parse %s: %w", rootPath, err)
	}
	if err = validateDoc(rootSch, rootMap, rootPath); err != nil {
		return nil, err
	}
	var root Root
	if err = yaml.Unmarshal(rootData, &root); err != nil {
		return nil, fmt.Errorf("config: decode root: %w", err)
	}
	if err = validateUniqueProviderIDs(&root, rootPath); err != nil {
		return nil, err
	}
	if err = validateDeclaredSchemaVersion(root.SchemaVersion, rootPath); err != nil {
		return nil, err
	}

	legacyRulesDir := filepath.Join(dir, "rules")
	if entries, lerr := os.ReadDir(legacyRulesDir); lerr == nil {
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			lower := strings.ToLower(e.Name())
			if strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml") {
				return nil, fmt.Errorf(
					"config: legacy rules/%s detected; migrate files to control/{states,routes,guards}/ with matching type",
					e.Name(),
				)
			}
		}
	} else if !os.IsNotExist(lerr) {
		return nil, fmt.Errorf("config: read rules dir: %w", lerr)
	}

	ruleFiles := make(map[string]RulesDocument)
	// Alphabetical kind order so RuleFiles keys sort as guards < routes < states (ADR-0011, ADR-0004).
	controlKinds := []string{"guards", "routes", "states"}
	for _, kind := range controlKinds {
		kindDir := filepath.Join(dir, "control", kind)
		entries, rerr := os.ReadDir(kindDir)
		if rerr != nil && !os.IsNotExist(rerr) {
			return nil, fmt.Errorf("config: read control/%s dir: %w", kind, rerr)
		}
		if rerr != nil {
			continue
		}
		var yamlNames []string
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			lower := strings.ToLower(e.Name())
			if strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml") {
				yamlNames = append(yamlNames, e.Name())
			}
		}
		sort.Strings(yamlNames)
		for _, name := range yamlNames {
			p := filepath.Join(kindDir, name)
			key := kind + "/" + name
			data, readErr := os.ReadFile(p)
			if readErr != nil {
				return nil, fmt.Errorf("config: read %s: %w", p, readErr)
			}
			var docMap map[string]any
			if uerr := yaml.Unmarshal(data, &docMap); uerr != nil {
				return nil, fmt.Errorf("config: parse %s: %w", p, uerr)
			}
			if err = validateDoc(rulesSch, docMap, p); err != nil {
				return nil, err
			}
			var doc RulesDocument
			if uerr := yaml.Unmarshal(data, &doc); uerr != nil {
				return nil, fmt.Errorf("config: decode %s: %w", p, uerr)
			}
			if err = validateRulesMatchControlKind(kind, doc.Rules, p); err != nil {
				return nil, err
			}
			ruleFiles[key] = doc
		}
	}

	res := &LoadResult{
		Dir:       dir,
		Root:      root,
		RuleFiles: ruleFiles,
	}

	kmPath := filepath.Join(dir, "knowledge", "manifest.json")
	kmData, err := os.ReadFile(kmPath)
	if err != nil {
		if os.IsNotExist(err) {
			return res, nil
		}
		return nil, fmt.Errorf("config: read knowledge manifest: %w", err)
	}
	var kmMap map[string]any
	if jerr := json.Unmarshal(kmData, &kmMap); jerr != nil {
		return nil, fmt.Errorf("config: parse knowledge manifest: %w", jerr)
	}
	if err = validateDoc(kmSch, kmMap, kmPath); err != nil {
		return nil, err
	}
	var km KnowledgeManifest
	if jerr := json.Unmarshal(kmData, &km); jerr != nil {
		return nil, fmt.Errorf("config: decode knowledge manifest: %w", jerr)
	}
	res.KnowledgePresent = true
	res.Knowledge = &km
	return res, nil
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
