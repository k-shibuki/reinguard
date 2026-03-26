package knowledge

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/k-shibuki/reinguard/internal/config"
)

// Hint thresholds for authoring discipline (ADR-0010); violations are warnings only.
const (
	MaxKnowledgeFileBytes = 256 * 1024
	MaxTriggersPerEntry   = 64
)

// HintWarnings returns human-readable warning lines (no trailing newline per line).
func HintWarnings(repoRootAbs string, m *config.KnowledgeManifest) []string {
	if m == nil {
		return nil
	}
	repoRootAbs = filepath.Clean(repoRootAbs)
	var w []string
	for _, e := range m.Entries {
		if len(e.Triggers) > MaxTriggersPerEntry {
			w = append(w, fmt.Sprintf("config warning: knowledge entry %q has %d triggers (consider reducing; max hint %d)",
				e.ID, len(e.Triggers), MaxTriggersPerEntry))
		}
		p := filepath.Join(repoRootAbs, filepath.FromSlash(e.Path))
		st, err := os.Stat(p)
		if err != nil {
			continue
		}
		if st.Size() > MaxKnowledgeFileBytes {
			w = append(w, fmt.Sprintf("config warning: knowledge file %q is large (%d bytes); consider splitting (hint max %d)",
				e.Path, st.Size(), MaxKnowledgeFileBytes))
		}
	}
	return w
}
