package schemaexport

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/k-shibuki/reinguard/pkg/schema"
)

func TestExport_writesFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := Export(dir); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(dir, schema.OperationalContextPlaceholder)
	st, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if st.Size() == 0 {
		t.Fatal("exported file empty")
	}
}

func TestListEmbedded(t *testing.T) {
	t.Parallel()
	names, err := ListEmbedded()
	if err != nil {
		t.Fatal(err)
	}
	if len(names) == 0 {
		t.Fatal("expected at least one embedded schema")
	}
}
