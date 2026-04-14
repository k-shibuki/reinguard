// Package procedure loads procedure markdown front matter under .reinguard/procedure (ADR-0011, Issue #117).
package procedure

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// AppliesTo mirrors procedure YAML front matter applies_to.
type AppliesTo struct {
	StateIDs []string `yaml:"state_ids"`
	RouteIDs []string `yaml:"route_ids"`
}

// FrontMatter is YAML metadata from the leading block in a procedure .md file.
//
//nolint:govet // fieldalignment: keep id/purpose/applies_to grouping for readability
type FrontMatter struct {
	ID        string    `yaml:"id"`
	Purpose   string    `yaml:"purpose"`
	AppliesTo AppliesTo `yaml:"applies_to"`
	// Other keys (reads, sense, act, …) are ignored for machine validation.
}

// ParseFrontMatter extracts and parses the first YAML front matter block (--- ... ---).
func ParseFrontMatter(md []byte) (*FrontMatter, error) {
	s := strings.TrimSpace(string(md))
	if !strings.HasPrefix(s, "---") {
		return nil, fmt.Errorf("procedure: missing opening front matter delimiter")
	}
	s = strings.TrimPrefix(s, "---")
	s = strings.TrimLeft(s, "\r\n")
	end := strings.Index(s, "\n---")
	if end < 0 {
		return nil, fmt.Errorf("procedure: missing closing front matter delimiter")
	}
	yamlBlock := strings.TrimSpace(s[:end])
	var fm FrontMatter
	if err := yaml.Unmarshal([]byte(yamlBlock), &fm); err != nil {
		return nil, fmt.Errorf("procedure: parse front matter yaml: %w", err)
	}
	fm.ID = strings.TrimSpace(fm.ID)
	fm.Purpose = strings.TrimSpace(fm.Purpose)
	fm.AppliesTo.StateIDs = trimNonEmptyStrings(fm.AppliesTo.StateIDs)
	fm.AppliesTo.RouteIDs = trimNonEmptyStrings(fm.AppliesTo.RouteIDs)
	if fm.ID == "" {
		return nil, fmt.Errorf("procedure: front matter: missing id")
	}
	if fm.Purpose == "" {
		return nil, fmt.Errorf("procedure: front matter: missing purpose")
	}
	if err := validateUniqueStrings("state_id", fm.ID, fm.AppliesTo.StateIDs); err != nil {
		return nil, err
	}
	if err := validateUniqueStrings("route_id", fm.ID, fm.AppliesTo.RouteIDs); err != nil {
		return nil, err
	}
	return &fm, nil
}

func trimNonEmptyStrings(in []string) []string {
	var out []string
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s != "" {
			out = append(out, s)
		}
	}
	return out
}

func validateUniqueStrings(kind, procedureID string, values []string) error {
	seen := make(map[string]string)
	for _, v := range values {
		key := v
		if prev, ok := seen[key]; ok {
			return fmt.Errorf("procedure: front matter: duplicate %s %q in procedure id %q (also %q)", kind, v, procedureID, prev)
		}
		seen[key] = v
	}
	return nil
}
