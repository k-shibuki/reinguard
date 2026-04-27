package rgdcli

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestRunStateEval_observationFile(t *testing.T) {
	t.Parallel()
	// Given: config with idle-on-main rule and observation JSON on main branch
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	writeFile(t, filepath.Join(cfgDir, "control", "states", "r.yaml"), []byte(testFixtureRulesStateIdle))
	obsDir := t.TempDir()
	writeFile(t, filepath.Join(obsDir, "o.json"), []byte(`{
  "schema_version": "0.8.0",
  "signals": {"git": {"branch": "main"}},
  "degraded": false
}`))

	var buf bytes.Buffer
	app := NewApp("test")
	app.Writer = &buf
	// When: state eval runs with --observation-file
	err := app.Run([]string{
		"rgd", "state", "eval",
		"--config-dir", cfgDir,
		"--observation-file", filepath.Join(obsDir, "o.json"),
	})
	if err != nil {
		t.Fatal(err)
	}
	// Then: resolved Idle state
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

func TestRunStateEval_traceRulesIncludesRuleTrace(t *testing.T) {
	t.Parallel()
	// Given: two state rules where only one matches; --trace-rules requested
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	writeFile(t, filepath.Join(cfgDir, "control", "states", "r.yaml"), []byte(`schema_version: "0.8.0"
rules:
  - type: state
    id: idle
    priority: 10
    state_id: Idle
    when:
      op: eq
      path: git.branch
      value: main
  - type: state
    id: feat
    priority: 10
    state_id: Working
    when:
      op: eq
      path: git.branch
      value: feat
`))
	obsDir := t.TempDir()
	writeFile(t, filepath.Join(obsDir, "o.json"), []byte(`{
  "schema_version": "0.8.0",
  "signals": {"git": {"branch": "main"}},
  "degraded": false
}`))

	var buf bytes.Buffer
	app := NewApp("t")
	app.Writer = &buf
	// When: state eval runs with --trace-rules
	if err := app.Run([]string{
		"rgd", "state", "eval",
		"--config-dir", cfgDir,
		"--observation-file", filepath.Join(obsDir, "o.json"),
		"--trace-rules",
	}); err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v; raw=%s", err, buf.String())
	}
	if out["kind"] != "resolved" || out["state_id"] != "Idle" {
		t.Fatalf("expected resolved Idle, got %+v", out)
	}
	// Then: rule_trace contains both evaluated rules in order with matched flags
	traceAny, ok := out["rule_trace"].([]any)
	if !ok {
		t.Fatalf("rule_trace missing or wrong type: %T (%v)", out["rule_trace"], out["rule_trace"])
	}
	if len(traceAny) != 2 {
		t.Fatalf("expected 2 trace entries, got %d: %+v", len(traceAny), traceAny)
	}
	for _, e := range traceAny {
		em, ok := e.(map[string]any)
		if !ok {
			t.Fatalf("trace entry has unexpected type %T: %v", e, e)
		}
		if em["rule_type"] != "state" {
			t.Fatalf("rule_type=%v, want state", em["rule_type"])
		}
		switch em["rule_id"] {
		case "idle":
			if em["matched"] != true {
				t.Fatalf("idle.matched=%v, want true", em["matched"])
			}
			if em["target_id"] != "Idle" {
				t.Fatalf("idle.target_id=%v", em["target_id"])
			}
		case "feat":
			if em["matched"] != false {
				t.Fatalf("feat.matched=%v, want false", em["matched"])
			}
		default:
			t.Fatalf("unexpected rule_id %v", em["rule_id"])
		}
	}
}

func TestRunStateEval_defaultOmitsRuleTrace(t *testing.T) {
	t.Parallel()
	// Given: same fixture as the trace test, but no --trace-rules flag
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	writeFile(t, filepath.Join(cfgDir, "control", "states", "r.yaml"), []byte(testFixtureRulesStateIdle))
	obsDir := t.TempDir()
	writeFile(t, filepath.Join(obsDir, "o.json"), []byte(`{"schema_version":"0.8.0","signals":{"git":{"branch":"main"}},"degraded":false}`))
	var buf bytes.Buffer
	app := NewApp("t")
	app.Writer = &buf
	// When: state eval runs without --trace-rules
	if err := app.Run([]string{
		"rgd", "state", "eval",
		"--config-dir", cfgDir,
		"--observation-file", filepath.Join(obsDir, "o.json"),
	}); err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatal(err)
	}
	// Then: rule_trace must be absent
	if _, ok := out["rule_trace"]; ok {
		t.Fatalf("default output must omit rule_trace, got %+v", out)
	}
}

func TestRunStateEval_traceRulesAmbiguousIncludesAllMatchedRules(t *testing.T) {
	t.Parallel()
	// Given: two same-priority state rules that both match the same observation
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	writeFile(t, filepath.Join(cfgDir, "control", "states", "r.yaml"), []byte(testFixtureRulesStateAmbiguous))
	obsDir := t.TempDir()
	writeFile(t, filepath.Join(obsDir, "o.json"), []byte(`{"schema_version":"0.8.0","signals":{"git":{"branch":"feat"}},"degraded":false}`))

	var buf bytes.Buffer
	app := NewApp("t")
	app.Writer = &buf
	// When: state eval runs with --trace-rules but without --fail-on-non-resolved
	if err := app.Run([]string{
		"rgd", "state", "eval",
		"--config-dir", cfgDir,
		"--observation-file", filepath.Join(obsDir, "o.json"),
		"--trace-rules",
	}); err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v; raw=%s", err, buf.String())
	}
	// Then: ambiguous outcome and both rules appear matched in rule_trace
	if out["kind"] != "ambiguous" {
		t.Fatalf("kind=%v, want ambiguous", out["kind"])
	}
	traceAny, ok := out["rule_trace"].([]any)
	if !ok || len(traceAny) != 2 {
		t.Fatalf("expected 2 trace entries, got %v", out["rule_trace"])
	}
	for _, e := range traceAny {
		em, ok := e.(map[string]any)
		if !ok {
			t.Fatalf("trace entry has unexpected type %T: %v", e, e)
		}
		if em["matched"] != true {
			t.Fatalf("ambiguous trace entry must be matched=true, got %+v", em)
		}
	}
}

func TestRunStateEval_unsupportedJSONOmitsEmptyFields(t *testing.T) {
	t.Parallel()
	// Given: same when as TestRunStateEval_failOnUnsupported (valid at config load; fails at match eval)
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	writeFile(t, filepath.Join(cfgDir, "control", "states", "bad.yaml"), []byte(`schema_version: "0.8.0"
rules:
  - type: state
    id: bad
    priority: 1
    state_id: X
    when:
      op: gt
      path: git.stash_count
      value: not-a-number
`))
	obsDir := t.TempDir()
	writeFile(t, filepath.Join(obsDir, "o.json"), []byte(`{"signals":{"git":{"stash_count": 0}}}`))
	var buf bytes.Buffer
	app := NewApp("t")
	app.Writer = &buf
	// When: state eval runs without --fail-on-non-resolved
	if err := app.Run([]string{
		"rgd", "state", "eval",
		"--config-dir", cfgDir,
		"--observation-file", filepath.Join(obsDir, "o.json"),
	}); err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("json: %v raw=%s", err, buf.String())
	}
	// Then: unsupported shape omits unset scalar fields (matches context build embedding)
	if out["kind"] != "unsupported" {
		t.Fatalf("kind=%v", out["kind"])
	}
	for _, k := range []string{"state_id", "route_id", "target_id", "rule_id", "priority"} {
		if _, ok := out[k]; ok {
			t.Fatalf("unexpected key %q in unsupported stdout: %v", k, out)
		}
	}
	if out["reason"] == nil || out["reason"] == "" {
		t.Fatalf("want non-empty reason, got %v", out["reason"])
	}
}
