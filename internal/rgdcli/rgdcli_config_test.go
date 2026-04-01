package rgdcli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunConfigValidate_invalidDir(t *testing.T) {
	t.Parallel()
	// Given: non-existent config-dir
	app := NewApp("t")
	app.Writer = &bytes.Buffer{}
	// When: config validate runs
	err := app.Run([]string{"rgd", "config", "validate", "--config-dir", filepath.Join(t.TempDir(), "missing-sub")})
	// Then: error
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunConfigValidate_deprecatedLegacyToolHints(t *testing.T) {
	t.Parallel()
	// Given: valid config with legacy_tool_hints set
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), []byte(`schema_version: "0.6.1"
default_branch: main
providers: []
legacy_tool_hints:
  note: "remove me"
`))
	var out, errBuf bytes.Buffer
	app := NewApp("t")
	app.Writer = &out
	app.ErrWriter = &errBuf
	// When: config validate runs
	if err := app.Run([]string{"rgd", "config", "validate", "--config-dir", dir}); err != nil {
		t.Fatal(err)
	}
	// Then: stderr warns about deprecation; stdout still reports OK
	if !strings.Contains(errBuf.String(), "legacy_tool_hints") || !strings.Contains(errBuf.String(), "deprecated") {
		t.Fatalf("stderr=%q", errBuf.String())
	}
	if !strings.Contains(out.String(), "config OK") {
		t.Fatalf("stdout=%q", out.String())
	}
}
