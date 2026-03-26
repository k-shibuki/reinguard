package rgdcli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRunSchemaExport_writesFiles(t *testing.T) {
	t.Parallel()
	// Given: empty export directory and CLI app writing to a buffer
	dir := filepath.Join(t.TempDir(), "schemas")

	var buf bytes.Buffer
	app := NewApp("test")
	app.Writer = &buf

	// When: schema export runs
	if err := app.Run([]string{"rgd", "schema", "export", "--dir", dir}); err != nil {
		t.Fatal(err)
	}
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	// Then: at least one file exists under dir
	if len(entries) == 0 {
		t.Fatal("expected schema files to be exported")
	}
}

func TestVersion_subcommand(t *testing.T) {
	t.Parallel()
	// Given: app with a fixed version string
	var buf bytes.Buffer
	app := NewApp("1.2.3")
	app.Writer = &buf

	// When: version subcommand runs
	if err := app.Run([]string{"rgd", "version"}); err != nil {
		t.Fatal(err)
	}
	// Then: stdout contains that version
	if !bytes.Contains(buf.Bytes(), []byte("1.2.3")) {
		t.Fatalf("expected version output, got %q", buf.String())
	}
}
