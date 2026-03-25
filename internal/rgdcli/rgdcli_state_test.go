package rgdcli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v; raw=%s", err, buf.String())
	}
	if out["kind"] != "resolved" {
		t.Fatalf("expected kind=resolved, got %v", out["kind"])
	}
	if out["state_id"] != "Idle" {
		t.Fatalf("expected state_id=Idle, got %v", out["state_id"])
	}
}

func TestRunStateEval_failOnNonResolved(t *testing.T) {
	t.Parallel()
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	if err := os.Mkdir(filepath.Join(cfgDir, "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(cfgDir, "rules", "r.yaml"), []byte(testFixtureRulesStateAmbiguous))
	obsDir := t.TempDir()
	writeFile(t, filepath.Join(obsDir, "o.json"), []byte(`{"signals":{"x":1}}`))
	app := NewApp("t")
	app.Writer = &bytes.Buffer{}
	err := app.Run([]string{
		"rgd", "state", "eval",
		"--config-dir", cfgDir,
		"--observation-file", filepath.Join(obsDir, "o.json"),
		"--fail-on-non-resolved",
	})
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("%v", err)
	}
}
