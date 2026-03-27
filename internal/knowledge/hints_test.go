package knowledge

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/k-shibuki/reinguard/internal/config"
	"github.com/k-shibuki/reinguard/pkg/schema"
)

func TestHintWarnings(t *testing.T) {
	t.Parallel()
	// Given/When/Then: each subtest uses tt.setup for disk+manifest, runs HintWarnings, expects tt.wantLen / wantPart
	tests := []struct {
		setup    func(t *testing.T) (root string, m *config.KnowledgeManifest)
		name     string
		wantPart string
		wantLen  int
	}{
		{
			setup: func(t *testing.T) (string, *config.KnowledgeManifest) {
				root := t.TempDir()
				m := &config.KnowledgeManifest{
					SchemaVersion: schema.CurrentSchemaVersion,
					Entries: []config.KnowledgeManifestEntry{
						{ID: "x", Path: "knowledge/x.md", Description: "d", Triggers: []string{"t"}},
					},
				}
				return root, m
			},
			name:     "ok_absent_file",
			wantPart: "",
			wantLen:  0,
		},
		{
			setup: func(t *testing.T) (string, *config.KnowledgeManifest) {
				root := t.TempDir()
				kdir := filepath.Join(root, "knowledge")
				if err := os.MkdirAll(kdir, 0o755); err != nil {
					t.Fatal(err)
				}
				p := filepath.Join(kdir, "edge.md")
				exact := make([]byte, MaxKnowledgeFileBytes)
				if err := os.WriteFile(p, exact, 0o644); err != nil {
					t.Fatal(err)
				}
				m := &config.KnowledgeManifest{
					SchemaVersion: schema.CurrentSchemaVersion,
					Entries: []config.KnowledgeManifestEntry{
						{ID: "x", Path: "knowledge/edge.md", Description: "d", Triggers: []string{"t"}},
					},
				}
				return root, m
			},
			name:     "file_at_max_bytes_no_warning",
			wantPart: "",
			wantLen:  0,
		},
		{
			setup: func(t *testing.T) (string, *config.KnowledgeManifest) {
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
				return root, m
			},
			name:     "file_over_max_bytes_warns",
			wantPart: "large",
			wantLen:  1,
		},
		{
			setup: func(t *testing.T) (string, *config.KnowledgeManifest) {
				root := t.TempDir()
				tr := make([]string, MaxTriggersPerEntry)
				for i := range tr {
					tr[i] = "t"
				}
				m := &config.KnowledgeManifest{
					SchemaVersion: schema.CurrentSchemaVersion,
					Entries: []config.KnowledgeManifestEntry{
						{ID: "x", Path: "knowledge/x.md", Description: "d", Triggers: tr},
					},
				}
				return root, m
			},
			name:     "triggers_at_max_no_warning",
			wantPart: "",
			wantLen:  0,
		},
		{
			setup: func(t *testing.T) (string, *config.KnowledgeManifest) {
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
				return root, m
			},
			name:     "triggers_over_max_warns",
			wantPart: "triggers",
			wantLen:  1,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			root, m := tt.setup(t)
			w := HintWarnings(root, m)
			if len(w) != tt.wantLen {
				t.Fatalf("got %d warnings %v", len(w), w)
			}
			if tt.wantLen > 0 && tt.wantPart != "" && !strings.Contains(w[0], tt.wantPart) {
				t.Fatalf("got %q", w[0])
			}
		})
	}
}
