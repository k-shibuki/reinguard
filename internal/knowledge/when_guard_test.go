package knowledge

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// Knowledge entries must not use state.state_id in when clauses: context build merges
// state after knowledge filtering would be circular; see docs/cli.md and ADR-0010.
func TestKnowledgeFrontMatterWhenDoesNotReferenceStateStateID(t *testing.T) {
	t.Parallel()
	repoRoot := filepath.Clean(filepath.Join("..", ".."))
	kdir := filepath.Join(repoRoot, ".reinguard", "knowledge")
	ents, err := os.ReadDir(kdir)
	if err != nil {
		t.Fatalf("read dir: %v", err)
	}
	for _, ent := range ents {
		if ent.IsDir() || !strings.HasSuffix(ent.Name(), ".md") {
			continue
		}
		p := filepath.Join(kdir, ent.Name())
		b, err := os.ReadFile(p)
		if err != nil {
			t.Fatalf("read %s: %v", p, err)
		}
		fm, ok := yamlFrontMatterOnly(b)
		if !ok {
			continue
		}
		if strings.Contains(string(fm), "state.state_id") {
			t.Fatalf("%s: knowledge when must not reference state.state_id (found in front matter)", p)
		}
	}
}

func yamlFrontMatterOnly(b []byte) ([]byte, bool) {
	if !bytes.HasPrefix(b, []byte("---\n")) && !bytes.HasPrefix(b, []byte("---\r\n")) {
		return nil, false
	}
	rest := b[3:]
	if bytes.HasPrefix(rest, []byte("\r\n")) {
		rest = rest[2:]
	}
	idx := bytes.Index(rest, []byte("\n---"))
	if idx < 0 {
		return nil, false
	}
	return rest[:idx], true
}
