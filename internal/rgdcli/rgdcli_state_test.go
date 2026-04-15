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
  "schema_version": "0.7.0",
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

func TestRunStateEval_unsupportedJSONOmitsEmptyFields(t *testing.T) {
	t.Parallel()
	// Given: same when as TestRunStateEval_failOnUnsupported (valid at config load; fails at match eval)
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	writeFile(t, filepath.Join(cfgDir, "control", "states", "bad.yaml"), []byte(`schema_version: "0.7.0"
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
