package rgdcli

import (
	"bytes"
	"encoding/json"
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
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON output: %v; raw=%s", err, buf.String())
	}
	if _, ok := out["paths"]; !ok {
		t.Fatalf("missing 'paths' key in output: %s", buf.String())
	}
}
