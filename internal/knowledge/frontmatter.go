// Package knowledge indexes repository knowledge files (ADR-0010).
package knowledge

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// FrontMatter is YAML metadata from the leading block in a knowledge .md file.
//
//nolint:govet // fieldalignment: keep id/description/triggers/when grouping for readability
type FrontMatter struct {
	ID          string   `yaml:"id"`
	Description string   `yaml:"description"`
	Triggers    []string `yaml:"triggers"`
	When        any      `yaml:"when"`
}

// ParseFrontMatter extracts and parses the first YAML front matter block (--- ... ---).
func ParseFrontMatter(md []byte) (*FrontMatter, error) {
	s := strings.TrimSpace(string(md))
	if !strings.HasPrefix(s, "---") {
		return nil, fmt.Errorf("knowledge: missing opening front matter delimiter")
	}
	s = strings.TrimPrefix(s, "---")
	s = strings.TrimLeft(s, "\r\n")
	end := strings.Index(s, "\n---")
	if end < 0 {
		return nil, fmt.Errorf("knowledge: missing closing front matter delimiter")
	}
	yamlBlock := strings.TrimSpace(s[:end])
	var fm FrontMatter
	if err := yaml.Unmarshal([]byte(yamlBlock), &fm); err != nil {
		return nil, fmt.Errorf("knowledge: parse front matter yaml: %w", err)
	}
	fm.ID = strings.TrimSpace(fm.ID)
	fm.Description = strings.TrimSpace(fm.Description)
	var triggers []string
	for _, t := range fm.Triggers {
		t = strings.TrimSpace(t)
		if t != "" {
			triggers = append(triggers, t)
		}
	}
	fm.Triggers = triggers
	if fm.ID == "" {
		return nil, fmt.Errorf("knowledge: front matter: missing id")
	}
	if fm.Description == "" {
		return nil, fmt.Errorf("knowledge: front matter: missing description")
	}
	if len(fm.Triggers) == 0 {
		return nil, fmt.Errorf("knowledge: front matter: triggers must have at least one non-empty entry")
	}
	if err := validateUniqueTriggers(fm.ID, fm.Triggers); err != nil {
		return nil, err
	}
	if fm.When == nil {
		return nil, fmt.Errorf("knowledge: front matter: missing required when")
	}
	if _, ok := fm.When.(map[string]any); !ok {
		if _, ok := fm.When.([]any); !ok {
			return nil, fmt.Errorf("knowledge: front matter: when must be object or array, got %T", fm.When)
		}
	}
	return &fm, nil
}

// validateUniqueTriggers rejects duplicate triggers after trim, compared case-insensitively.
func validateUniqueTriggers(entryID string, triggers []string) error {
	seen := make(map[string]string)
	for _, t := range triggers {
		key := strings.ToLower(t)
		if prev, ok := seen[key]; ok {
			return fmt.Errorf("knowledge: front matter: duplicate trigger %q (also %q) in id %q", t, prev, entryID)
		}
		seen[key] = t
	}
	return nil
}
