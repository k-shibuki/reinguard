package knowledge

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/k-shibuki/reinguard/internal/config"
	"github.com/k-shibuki/reinguard/pkg/schema"
)

// BuildManifest scans knowledgeAbsDir for *.md, parses front matter, and builds a manifest.
// Paths in entries are repo-relative with forward slashes.
func BuildManifest(repoRootAbs, knowledgeAbsDir string) (*config.KnowledgeManifest, error) {
	entries, err := indexEntries(repoRootAbs, knowledgeAbsDir)
	if err != nil {
		return nil, err
	}
	return &config.KnowledgeManifest{
		SchemaVersion: schema.CurrentSchemaVersion,
		Entries:       entries,
	}, nil
}

func indexEntries(repoRootAbs, knowledgeAbsDir string) ([]config.KnowledgeManifestEntry, error) {
	repoRootAbs = filepath.Clean(repoRootAbs)
	knowledgeAbsDir = filepath.Clean(knowledgeAbsDir)
	de, err := os.ReadDir(knowledgeAbsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("knowledge: directory %q does not exist", knowledgeAbsDir)
		}
		return nil, fmt.Errorf("knowledge: read directory: %w", err)
	}
	var names []string
	for _, e := range de {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.EqualFold(filepath.Ext(name), ".md") {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)

	seenID := make(map[string]string)
	var entries []config.KnowledgeManifestEntry
	for _, name := range names {
		absPath := filepath.Join(knowledgeAbsDir, name)
		data, rerr := os.ReadFile(absPath)
		if rerr != nil {
			return nil, fmt.Errorf("knowledge: read %s: %w", absPath, rerr)
		}
		fm, perr := ParseFrontMatter(data)
		if perr != nil {
			return nil, fmt.Errorf("%s: %w", absPath, perr)
		}
		if prev, ok := seenID[fm.ID]; ok {
			return nil, fmt.Errorf("knowledge: duplicate id %q in %s and %s", fm.ID, prev, absPath)
		}
		seenID[fm.ID] = absPath

		rel, rerr := filepath.Rel(repoRootAbs, absPath)
		if rerr != nil {
			return nil, fmt.Errorf("knowledge: rel path: %w", rerr)
		}
		rel = filepath.ToSlash(rel)

		entries = append(entries, config.KnowledgeManifestEntry{
			ID:          fm.ID,
			Path:        rel,
			Description: fm.Description,
			Triggers:    append([]string(nil), fm.Triggers...),
			When:        fm.When,
		})
	}
	return entries, nil
}
