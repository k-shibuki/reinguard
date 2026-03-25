package rgdcli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestStringField(t *testing.T) {
	t.Parallel()
	m := map[string]any{"name": "alice", "count": 42}
	if got := stringField(m, "name"); got != "alice" {
		t.Fatalf("got %q", got)
	}
	if got := stringField(m, "count"); got != "" {
		t.Fatalf("expected empty for non-string, got %q", got)
	}
	if got := stringField(m, "missing"); got != "" {
		t.Fatalf("expected empty for missing key, got %q", got)
	}
}

func TestRunSchemaExport_writesFiles(t *testing.T) {
	t.Parallel()
	dir := filepath.Join(t.TempDir(), "schemas")

	var buf bytes.Buffer
	app := NewApp("test")
	app.Writer = &buf

	if err := app.Run([]string{"rgd", "schema", "export", "--dir", dir}); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(entries) == 0 {
		t.Fatal("expected schema files to be exported")
	}
}

func TestVersion_subcommand(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	app := NewApp("1.2.3")
	app.Writer = &buf

	if err := app.Run([]string{"rgd", "version"}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf.Bytes(), []byte("1.2.3")) {
		t.Fatalf("expected version output, got %q", buf.String())
	}
}
