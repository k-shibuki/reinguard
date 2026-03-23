package rgdcli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestRunStateEval_observationFile(t *testing.T) {
	t.Parallel()
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	if err := os.Mkdir(filepath.Join(cfgDir, "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(cfgDir, "rules", "r.yaml"), []byte(testFixtureRulesStateIdle))
	obsDir := t.TempDir()
	writeFile(t, filepath.Join(obsDir, "o.json"), []byte(`{
  "schema_version": "0.2.0",
  "signals": {"git": {"branch": "main"}},
  "degraded": false
}`))

	var buf bytes.Buffer
	app := NewApp("test")
	app.Writer = &buf
	err := app.Run([]string{
		"rgd", "state", "eval",
		"--config-dir", cfgDir,
		"--observation-file", filepath.Join(obsDir, "o.json"),
	})
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"kind"`)) {
		t.Fatalf("%s", buf.String())
	}
}
