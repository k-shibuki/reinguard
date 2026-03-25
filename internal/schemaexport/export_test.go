package schemaexport

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/k-shibuki/reinguard/pkg/schema"
)

func TestExport_writesFile(t *testing.T) {
	t.Parallel()
	// Given: empty temp directory
	dir := t.TempDir()
	// When: Export runs
	if err := Export(dir); err != nil {
		t.Fatal(err)
	}
	// Then: operational context schema file is non-empty
	p := filepath.Join(dir, schema.OperationalContext)
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
