// Package config loads and validates .reinguard configuration (ADR-0008).
package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
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

// Rule is one rule from rules/*.yaml.
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

// KnowledgeManifest is .reinguard/knowledge/manifest.json.
type KnowledgeManifest struct {
	SchemaVersion string `json:"schema_version"`
	Entries       []struct {
		ID   string `json:"id"`
		Path string `json:"path"`
	} `json:"entries"`
}

// LoadResult holds validated configuration from a config directory.
type LoadResult struct {
	RuleFiles        map[string]RulesDocument
	Knowledge        *KnowledgeManifest
	Dir              string
	Root             Root
	KnowledgePresent bool
}

// Load reads reinguard.yaml, all rules/*.yaml, and optional knowledge/manifest.json.
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

	ruleFiles := make(map[string]RulesDocument)
	rulesDir := filepath.Join(dir, "rules")
	entries, err := os.ReadDir(rulesDir)
	if err != nil && !os.IsNotExist(err) {
		return nil, fmt.Errorf("config: read rules dir: %w", err)
	}
	if err == nil {
		var yamlNames []string
		for _, e := range entries {
			if e.IsDir() {
				continue
			}
			if strings.HasSuffix(strings.ToLower(e.Name()), ".yaml") || strings.HasSuffix(strings.ToLower(e.Name()), ".yml") {
				yamlNames = append(yamlNames, e.Name())
			}
		}
		sort.Strings(yamlNames)
		for _, name := range yamlNames {
			p := filepath.Join(rulesDir, name)
			data, rerr := os.ReadFile(p)
			if rerr != nil {
				return nil, fmt.Errorf("config: read %s: %w", p, rerr)
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
			ruleFiles[name] = doc
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

// Rules returns all rules from all files in stable sort order by filename then index.
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

// DeprecatedWarnings returns human-readable stderr lines for deprecated fields (ADR-0008).
func DeprecatedWarnings(root *Root) []string {
	if root == nil || len(root.LegacyToolHints) == 0 {
		return nil
	}
	return []string{
		`config warning: "legacy_tool_hints" is deprecated; remove it from reinguard.yaml (see JSON Schema / docs/cli.md)`,
	}
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
