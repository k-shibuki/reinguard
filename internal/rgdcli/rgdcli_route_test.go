package rgdcli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunRouteSelect_failOnNonResolved(t *testing.T) {
	t.Parallel()
	// Given: ambiguous route rules and matching signals
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	writeFile(t, filepath.Join(cfgDir, "control", "routes", "r.yaml"), []byte(testFixtureRulesRouteAmbiguous))
	obsDir := t.TempDir()
	writeFile(t, filepath.Join(obsDir, "o.json"), []byte(`{"signals":{"x":1},"degraded":false}`))
	var buf bytes.Buffer
	app := NewApp("t")
	app.Writer = &buf
	// When: route select runs with --fail-on-non-resolved
	err := app.Run([]string{
		"rgd", "route", "select",
		"--config-dir", cfgDir,
		"--observation-file", filepath.Join(obsDir, "o.json"),
		"--fail-on-non-resolved",
	})
	// Then: error mentions ambiguity
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("%v / %s", err, buf.String())
	}
}

func TestRunRouteSelect_stateFileFlattensStateDottedKeys(t *testing.T) {
	t.Parallel()
	// Given: a route rule that matches dotted path state.kind
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	writeFile(t, filepath.Join(cfgDir, "control", "routes", "r.yaml"), []byte(testFixtureControlRoutesNext))
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

func TestRunRouteSelect_relativeObservationAndStateFileWithCwd(t *testing.T) {
	t.Parallel()
	// Given: data dir with relative observation/state files and --cwd
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	writeFile(t, filepath.Join(cfgDir, "control", "routes", "r.yaml"), []byte(testFixtureControlRoutesNext))
	dataDir := t.TempDir()
	writeFile(t, filepath.Join(dataDir, "o.json"), []byte(`{"signals":{"x":1},"degraded":false}`))
	writeFile(t, filepath.Join(dataDir, "s.json"), []byte(`{"kind":"resolved","state_id":"Idle"}`))

	var buf bytes.Buffer
	app := NewApp("t")
	app.Writer = &buf
	// When: route select resolves paths against cwd
	err := app.Run([]string{
		"rgd", "route", "select",
		"--config-dir", cfgDir,
		"--cwd", dataDir,
		"--observation-file", "o.json",
		"--state-file", "s.json",
	})
	if err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON output: %v; raw=%s", err, buf.String())
	}
	// Then: resolved route next
	if out["kind"] != "resolved" || out["route_id"] != "next" {
		t.Fatalf("unexpected route output: %v", out)
	}
}
