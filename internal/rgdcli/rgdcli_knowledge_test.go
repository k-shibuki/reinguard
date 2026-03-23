package rgdcli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRunKnowledgePack_emptyManifest(t *testing.T) {
	t.Parallel()
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	if err := os.Mkdir(filepath.Join(cfgDir, "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	var buf bytes.Buffer
	app := NewApp("t")
	app.Writer = &buf
	if err := app.Run([]string{"rgd", "knowledge", "pack", "--config-dir", cfgDir}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"paths"`)) {
		t.Fatalf("%s", buf.String())
	}
}
