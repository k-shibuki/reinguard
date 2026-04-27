package rgdcli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestRunRouteSelect_stateFileFlattensStateDottedKeys(t *testing.T) {
	t.Parallel()
	// Given: a route rule that matches dotted path state.kind
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	writeFile(t, filepath.Join(cfgDir, "control", "routes", "r.yaml"), []byte(testFixtureControlRoutesNext))
	obsDir := t.TempDir()
	writeFile(t, filepath.Join(obsDir, "o.json"), []byte(`{"signals":{"git":{"branch":"main"}},"degraded":false}`))
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

func TestRunRouteSelect_traceRulesIncludesRuleTrace(t *testing.T) {
	t.Parallel()
	// Given: route rules where one matches state.kind=resolved and one does not
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	writeFile(t, filepath.Join(cfgDir, "control", "routes", "r.yaml"), []byte(`schema_version: "0.8.0"
rules:
  - type: route
    id: r1
    priority: 10
    route_id: next
    when:
      op: eq
      path: state.kind
      value: resolved
  - type: route
    id: r2
    priority: 5
    route_id: other
    when:
      op: eq
      path: state.kind
      value: degraded
`))
	obsDir := t.TempDir()
	writeFile(t, filepath.Join(obsDir, "o.json"), []byte(`{"signals":{"git":{"branch":"main"}},"degraded":false}`))
	stateDir := t.TempDir()
	writeFile(t, filepath.Join(stateDir, "s.json"), []byte(`{"kind":"resolved","state_id":"Idle"}`))

	var buf bytes.Buffer
	app := NewApp("t")
	app.Writer = &buf
	// When: route select runs with --trace-rules
	if err := app.Run([]string{
		"rgd", "route", "select",
		"--config-dir", cfgDir,
		"--observation-file", filepath.Join(obsDir, "o.json"),
		"--state-file", filepath.Join(stateDir, "s.json"),
		"--trace-rules",
	}); err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v; raw=%s", err, buf.String())
	}
	if out["route_id"] != "next" {
		t.Fatalf("route_id=%v, want next", out["route_id"])
	}
	traceAny, ok := out["rule_trace"].([]any)
	if !ok || len(traceAny) != 2 {
		t.Fatalf("expected 2 trace entries, got %v", out["rule_trace"])
	}
	// Then: matched flag agrees with route resolution
	for _, e := range traceAny {
		em, ok := e.(map[string]any)
		if !ok {
			t.Fatalf("trace entry has unexpected type %T: %v", e, e)
		}
		if em["rule_type"] != "route" {
			t.Fatalf("rule_type=%v, want route", em["rule_type"])
		}
		switch em["rule_id"] {
		case "r1":
			if em["matched"] != true {
				t.Fatalf("r1.matched=%v, want true", em["matched"])
			}
			if em["target_id"] != "next" {
				t.Fatalf("r1.target_id=%v", em["target_id"])
			}
		case "r2":
			if em["matched"] != false {
				t.Fatalf("r2.matched=%v, want false", em["matched"])
			}
		default:
			t.Fatalf("unexpected rule_id %v", em["rule_id"])
		}
	}
}

func TestRunRouteSelect_defaultOmitsRuleTrace(t *testing.T) {
	t.Parallel()
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	writeFile(t, filepath.Join(cfgDir, "control", "routes", "r.yaml"), []byte(testFixtureControlRoutesNext))
	obsDir := t.TempDir()
	writeFile(t, filepath.Join(obsDir, "o.json"), []byte(`{"signals":{"git":{"branch":"main"}},"degraded":false}`))
	stateDir := t.TempDir()
	writeFile(t, filepath.Join(stateDir, "s.json"), []byte(`{"kind":"resolved","state_id":"Idle"}`))

	var buf bytes.Buffer
	app := NewApp("t")
	app.Writer = &buf
	// When: route select runs without --trace-rules
	if err := app.Run([]string{
		"rgd", "route", "select",
		"--config-dir", cfgDir,
		"--observation-file", filepath.Join(obsDir, "o.json"),
		"--state-file", filepath.Join(stateDir, "s.json"),
	}); err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	// Then: rule_trace must be absent
	if _, ok := out["rule_trace"]; ok {
		t.Fatalf("default route select must omit rule_trace, got %+v", out)
	}
}

func TestRunRouteSelect_relativeObservationAndStateFileWithCwd(t *testing.T) {
	t.Parallel()
	// Given: data dir with relative observation/state files and --cwd
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	writeFile(t, filepath.Join(cfgDir, "control", "routes", "r.yaml"), []byte(testFixtureControlRoutesNext))
	dataDir := t.TempDir()
	writeFile(t, filepath.Join(dataDir, "o.json"), []byte(`{"signals":{"git":{"branch":"main"}},"degraded":false}`))
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
