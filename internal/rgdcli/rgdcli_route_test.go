package rgdcli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunRouteSelect_failOnNonResolved(t *testing.T) {
	t.Parallel()
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	if err := os.Mkdir(filepath.Join(cfgDir, "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(cfgDir, "rules", "r.yaml"), []byte(testFixtureRulesRouteAmbiguous))
	obsDir := t.TempDir()
	writeFile(t, filepath.Join(obsDir, "o.json"), []byte(`{"signals":{"x":1},"degraded":false}`))
	var buf bytes.Buffer
	app := NewApp("t")
	app.Writer = &buf
	err := app.Run([]string{
		"rgd", "route", "select",
		"--config-dir", cfgDir,
		"--observation-file", filepath.Join(obsDir, "o.json"),
		"--fail-on-non-resolved",
	})
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("%v / %s", err, buf.String())
	}
}

func TestRunRouteSelect_stateFileFlattensStateDottedKeys(t *testing.T) {
	t.Parallel()
	// Given: a route rule that matches dotted path state.kind
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	if err := os.Mkdir(filepath.Join(cfgDir, "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(cfgDir, "rules", "r.yaml"), []byte(`rules:
  - type: route
    id: by_state
    priority: 10
    route_id: next
    when:
      op: eq
      path: state.kind
      value: resolved
`))
	obsDir := t.TempDir()
	writeFile(t, filepath.Join(obsDir, "o.json"), []byte(`{"signals":{"x":1},"degraded":false}`))
	stateDir := t.TempDir()
	writeFile(t, filepath.Join(stateDir, "s.json"), []byte(`{"kind":"resolved","state_id":"Idle"}`))

	// When: route select runs with --state-file
	var buf bytes.Buffer
	app := NewApp("t")
	app.Writer = &buf
	err := app.Run([]string{
		"rgd", "route", "select",
		"--config-dir", cfgDir,
		"--observation-file", filepath.Join(obsDir, "o.json"),
		"--state-file", filepath.Join(stateDir, "s.json"),
	})
	if err != nil {
		t.Fatal(err)
	}

	// Then: route resolves via state.kind
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON output: %v; raw=%s", err, buf.String())
	}
	if out["kind"] != "resolved" || out["route_id"] != "next" {
		t.Fatalf("unexpected route output: %v", out)
	}
}
