package knowledge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/k-shibuki/reinguard/internal/config"
	"github.com/k-shibuki/reinguard/pkg/schema"
)

func TestHintWarnings_ok(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	m := &config.KnowledgeManifest{
		SchemaVersion: schema.CurrentSchemaVersion,
		Entries: []config.KnowledgeManifestEntry{
			{ID: "x", Path: "knowledge/x.md", Description: "d", Triggers: []string{"t"}},
		},
	}
	w := HintWarnings(root, m)
	if len(w) != 0 {
		t.Fatalf("%v", w)
	}
}

func TestHintWarnings_oversizedFile(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	kdir := filepath.Join(root, "knowledge")
	if err := os.MkdirAll(kdir, 0o755); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(kdir, "big.md")
	large := make([]byte, MaxKnowledgeFileBytes+1)
	if err := os.WriteFile(p, large, 0o644); err != nil {
		t.Fatal(err)
	}
	m := &config.KnowledgeManifest{
		SchemaVersion: schema.CurrentSchemaVersion,
		Entries: []config.KnowledgeManifestEntry{
			{ID: "x", Path: "knowledge/big.md", Description: "d", Triggers: []string{"t"}},
		},
	}
	w := HintWarnings(root, m)
	if len(w) != 1 || !strings.Contains(w[0], "large") {
		t.Fatalf("%v", w)
	}
}

func TestHintWarnings_manyTriggers(t *testing.T) {
	t.Parallel()
	root := t.TempDir()
	tr := make([]string, MaxTriggersPerEntry+1)
	for i := range tr {
		tr[i] = "t"
	}
	m := &config.KnowledgeManifest{
		SchemaVersion: schema.CurrentSchemaVersion,
		Entries: []config.KnowledgeManifestEntry{
			{ID: "x", Path: "knowledge/x.md", Description: "d", Triggers: tr},
		},
	}
	w := HintWarnings(root, m)
	if len(w) != 1 || !strings.Contains(w[0], "triggers") {
		t.Fatalf("%v", w)
	}
}
