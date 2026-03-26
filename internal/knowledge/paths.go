package knowledge

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/k-shibuki/reinguard/internal/config"
)

// ValidateEntryPaths ensures each manifest entry path exists under repoRootAbs.
func ValidateEntryPaths(repoRootAbs string, m *config.KnowledgeManifest) error {
	if m == nil {
		return nil
	}
	repoRootAbs = filepath.Clean(repoRootAbs)
	for _, e := range m.Entries {
		p := filepath.Join(repoRootAbs, filepath.FromSlash(e.Path))
		st, err := os.Stat(p)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Errorf("knowledge: manifest entry %q path does not exist: %s", e.ID, e.Path)
			}
			return fmt.Errorf("knowledge: manifest entry %q path %q: %w", e.ID, e.Path, err)
		}
		if st.IsDir() {
			return fmt.Errorf("knowledge: manifest entry %q path is a directory: %s", e.ID, e.Path)
		}
	}
	return nil
}
