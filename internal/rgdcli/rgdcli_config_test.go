package rgdcli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunConfigValidate_invalidDir(t *testing.T) {
	t.Parallel()
	app := NewApp("t")
	app.Writer = &bytes.Buffer{}
	err := app.Run([]string{"rgd", "config", "validate", "--config-dir", filepath.Join(t.TempDir(), "missing-sub")})
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestRunConfigValidate_deprecatedLegacyToolHints(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "reinguard.yaml"), []byte(`schema_version: "0.3.0"
default_branch: main
providers: []
legacy_tool_hints:
  note: "remove me"
`))
	var out, errBuf bytes.Buffer
	app := NewApp("t")
	app.Writer = &out
	app.ErrWriter = &errBuf
	if err := app.Run([]string{"rgd", "config", "validate", "--config-dir", dir}); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(errBuf.String(), "legacy_tool_hints") || !strings.Contains(errBuf.String(), "deprecated") {
		t.Fatalf("stderr=%q", errBuf.String())
	}
	if !strings.Contains(out.String(), "config OK") {
		t.Fatalf("stdout=%q", out.String())
	}
}
