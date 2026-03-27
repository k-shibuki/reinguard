// Package labels defines GitHub label metadata from .reinguard/labels.yaml (ADR-0008).
package labels

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"gopkg.in/yaml.v3"
)

//go:embed data/labels.yaml
var embeddedLabelsYAML []byte

// TypeLabels is the set of PR/Issue type label names (scope shared). Built from embedded labels.yaml at init.
var TypeLabels map[string]struct{}

func init() {
	c, err := LoadFromBytes(embeddedLabelsYAML)
	if err != nil {
		panic("labels: embedded .reinguard/labels.yaml: " + err.Error())
	}
	TypeLabels = c.TypeLabelsMap()
}

// TypeLabelNames returns sorted type label names from the embedded labels.yaml (stable for tests).
func TypeLabelNames() []string {
	c, err := LoadFromBytes(embeddedLabelsYAML)
	if err != nil {
		return nil
	}
	return c.TypeLabelNames()
}

// RepoLabel is a label to ensure or sync on the current gh repo (idempotent).
type RepoLabel struct {
	Name        string
	Color       string
	Description string
}

// Config is the parsed .reinguard/labels.yaml document.
type Config struct {
	Categories    map[string]Category `yaml:"categories"`
	SchemaVersion string              `yaml:"schema_version"`
}

// Category is one entry under categories (e.g. type, exception, scope).
type Category struct {
	Description string       `yaml:"description"`
	Scope       string       `yaml:"scope"`
	Labels      []LabelEntry `yaml:"labels"`
}

// LabelEntry is one label specification.
type LabelEntry struct {
	CommitPrefix *bool  `yaml:"commit_prefix,omitempty"`
	Name         string `yaml:"name"`
	Color        string `yaml:"color"`
	Description  string `yaml:"description"`
}

// LoadFromBytes unmarshals labels YAML (no schema validation; use config.Load for that).
func LoadFromBytes(data []byte) (*Config, error) {
	var c Config
	if err := yaml.Unmarshal(data, &c); err != nil {
		return nil, fmt.Errorf("labels: parse yaml: %w", err)
	}
	return &c, nil
}

// LoadFromFile reads and unmarshals labels.yaml (no schema validation; use config.Load for that).
func LoadFromFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("labels: read %s: %w", path, err)
	}
	return LoadFromBytes(data)
}

// LoadFromConfigDir loads .reinguard/labels.yaml under dir (config directory root).
func LoadFromConfigDir(dir string) (*Config, error) {
	return LoadFromFile(filepath.Join(dir, "labels.yaml"))
}

// TypeLabelNames returns sorted type label names (categories.type, scope shared).
func (c *Config) TypeLabelNames() []string {
	if c == nil || c.Categories == nil {
		return nil
	}
	cat, ok := c.Categories["type"]
	if !ok || cat.Scope != "shared" {
		return nil
	}
	var names []string
	for _, e := range cat.Labels {
		names = append(names, e.Name)
	}
	sort.Strings(names)
	return names
}

// ExceptionLabelNames returns sorted exception label names (categories.exception).
func (c *Config) ExceptionLabelNames() []string {
	if c == nil || c.Categories == nil {
		return nil
	}
	cat, ok := c.Categories["exception"]
	if !ok {
		return nil
	}
	var names []string
	for _, e := range cat.Labels {
		names = append(names, e.Name)
	}
	sort.Strings(names)
	return names
}

// IssueOnlyLabelNames returns sorted scope/issue-only label names (categories.scope).
func (c *Config) IssueOnlyLabelNames() []string {
	if c == nil || c.Categories == nil {
		return nil
	}
	cat, ok := c.Categories["scope"]
	if !ok {
		return nil
	}
	var names []string
	for _, e := range cat.Labels {
		names = append(names, e.Name)
	}
	sort.Strings(names)
	return names
}

// CommitPrefixNames returns sorted label names with commit_prefix == true.
func (c *Config) CommitPrefixNames() []string {
	if c == nil || c.Categories == nil {
		return nil
	}
	var names []string
	for _, cat := range c.Categories {
		for _, e := range cat.Labels {
			if e.CommitPrefix != nil && *e.CommitPrefix {
				names = append(names, e.Name)
			}
		}
	}
	sort.Strings(names)
	return names
}

// AllRepoLabels flattens all categories into RepoLabel rows for ensure/sync.
func (c *Config) AllRepoLabels() []RepoLabel {
	if c == nil || c.Categories == nil {
		return nil
	}
	var keys []string
	for k := range c.Categories {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var out []RepoLabel
	for _, k := range keys {
		for _, e := range c.Categories[k].Labels {
			out = append(out, RepoLabel{
				Name:        e.Name,
				Color:       strings.TrimPrefix(strings.ToUpper(e.Color), "#"),
				Description: e.Description,
			})
		}
	}
	return out
}

// TypeLabelsMap returns a set map of type label names (shared scope) for PR/Issue policy checks.
func (c *Config) TypeLabelsMap() map[string]struct{} {
	m := make(map[string]struct{})
	for _, n := range c.TypeLabelNames() {
		m[n] = struct{}{}
	}
	return m
}

// LabelsByName builds name -> LabelEntry across all categories (last wins if duplicate).
func (c *Config) LabelsByName() map[string]LabelEntry {
	if c == nil || c.Categories == nil {
		return nil
	}
	out := make(map[string]LabelEntry)
	for _, cat := range c.Categories {
		for _, e := range cat.Labels {
			out[e.Name] = e
		}
	}
	return out
}
