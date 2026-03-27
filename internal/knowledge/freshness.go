package knowledge

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"reflect"
	"sort"

	"github.com/k-shibuki/reinguard/internal/config"
	"github.com/k-shibuki/reinguard/pkg/schema"
)

// CheckFreshness returns an error if m does not match a manifest built from knowledgeAbsDir.
func CheckFreshness(m *config.KnowledgeManifest, repoRootAbs, knowledgeAbsDir string) error {
	if m == nil {
		return nil
	}
	built, err := BuildManifest(repoRootAbs, knowledgeAbsDir)
	if err != nil {
		return err
	}
	if !reflect.DeepEqual(normalizeManifest(m), normalizeManifest(built)) {
		return fmt.Errorf("knowledge: manifest.json is stale relative to knowledge/*.md front matter; run: rgd knowledge index")
	}
	return nil
}

// normalizedManifest is a compare-friendly view of a knowledge manifest (sorted entries, stable paths).
type normalizedManifest struct {
	schemaVersion string
	entries       []normalizedEntry
}

// normalizedEntry is one manifest row after normalization for DeepEqual checks.
//
//nolint:govet // fieldalignment: mirror manifest field grouping
type normalizedEntry struct {
	id          string
	path        string
	description string
	triggers    []string
	when        any
}

// normalizeManifest copies m into a canonical form so two manifests can be compared with reflect.DeepEqual.
func normalizeManifest(m *config.KnowledgeManifest) normalizedManifest {
	entries := make([]normalizedEntry, 0, len(m.Entries))
	for _, e := range m.Entries {
		tr := append([]string(nil), e.Triggers...)
		sort.Strings(tr)
		entries = append(entries, normalizedEntry{
			id:          e.ID,
			path:        filepath.ToSlash(e.Path),
			description: e.Description,
			triggers:    tr,
			when:        canonicalizeWhenForCompare(e.When),
		})
	}
	sort.Slice(entries, func(i, j int) bool {
		if entries[i].path != entries[j].path {
			return entries[i].path < entries[j].path
		}
		return entries[i].id < entries[j].id
	})
	sv := m.SchemaVersion
	if sv == "" {
		sv = schema.CurrentSchemaVersion
	}
	return normalizedManifest{schemaVersion: sv, entries: entries}
}

// canonicalizeWhenForCompare round-trips when through JSON so YAML vs JSON numeric shapes compare equal.
func canonicalizeWhenForCompare(v any) any {
	if v == nil {
		return nil
	}
	b, err := json.Marshal(v)
	if err != nil {
		return v
	}
	var out any
	if err := json.Unmarshal(b, &out); err != nil {
		return v
	}
	return out
}
