package rgdcli

import (
	"bytes"
	"encoding/json"
	"os/exec"
	"path/filepath"
	"testing"
)

func TestRunGateRecordStatusShow_roundTrip(t *testing.T) {
	t.Parallel()
	// Given: a git repo on main and a checks file for local verification
	repo := initGitRepoForGateCLI(t)
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(repo, "checks.json"), []byte(`[
  {"id":"go-test","status":"pass","summary":"go test ./... -race"},
  {"id":"golangci-lint","status":"pass","summary":"golangci-lint run"}
]`))

	// When: gate record, status, and show run through the CLI
	var recordBuf bytes.Buffer
	app := NewApp("t")
	app.Writer = &recordBuf
	if err := app.Run([]string{
		"rgd", "gate", "record",
		"--config-dir", cfgDir,
		"--cwd", repo,
		"--status", "pass",
		"--checks-file", "checks.json",
		"local-verification",
	}); err != nil {
		t.Fatal(err)
	}
	var statusBuf bytes.Buffer
	app2 := NewApp("t")
	app2.Writer = &statusBuf
	if err := app2.Run([]string{
		"rgd", "gate", "status",
		"--config-dir", cfgDir,
		"--cwd", repo,
		"local-verification",
	}); err != nil {
		t.Fatal(err)
	}
	var showBuf bytes.Buffer
	app3 := NewApp("t")
	app3.Writer = &showBuf
	if err := app3.Run([]string{
		"rgd", "gate", "show",
		"--config-dir", cfgDir,
		"local-verification",
	}); err != nil {
		t.Fatal(err)
	}

	// Then: the artifact persists and status resolves to pass on the same HEAD
	var recordOut map[string]any
	if err := json.Unmarshal(recordBuf.Bytes(), &recordOut); err != nil {
		t.Fatalf("record json: %v raw=%s", err, recordBuf.String())
	}
	if recordOut["gate_id"] != "local-verification" || recordOut["status"] != "pass" {
		t.Fatalf("record output=%v", recordOut)
	}
	var statusOut map[string]any
	if err := json.Unmarshal(statusBuf.Bytes(), &statusOut); err != nil {
		t.Fatalf("status json: %v raw=%s", err, statusBuf.String())
	}
	if statusOut["status"] != "pass" {
		t.Fatalf("status output=%v", statusOut)
	}
	var showOut map[string]any
	if err := json.Unmarshal(showBuf.Bytes(), &showOut); err != nil {
		t.Fatalf("show json: %v raw=%s", err, showBuf.String())
	}
	checks, ok := showOut["checks"].([]any)
	if !ok || len(checks) != 2 {
		t.Fatalf("show output=%v", showOut)
	}
}

func TestRunStateEval_mergesRuntimeGateSignals(t *testing.T) {
	t.Parallel()
	// Given: a state rule keyed on gates.local-verification.status and a recorded artifact
	repo := initGitRepoForGateCLI(t)
	cfgDir := filepath.Join(repo, ".reinguard")
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	writeFile(t, filepath.Join(cfgDir, "control", "states", "gates.yaml"), []byte(`rules:
  - type: state
    id: local_verified
    priority: 1
    state_id: ready_for_pr
    when:
      op: eq
      path: gates.local-verification.status
      value: pass
`))
	writeFile(t, filepath.Join(repo, "obs.json"), []byte(`{"signals":{"git":{"branch":"main"}},"degraded":false}`))

	var recordBuf bytes.Buffer
	app := NewApp("t")
	app.Writer = &recordBuf
	if err := app.Run([]string{
		"rgd", "gate", "record",
		"--config-dir", cfgDir,
		"--cwd", repo,
		"--status", "pass",
		"local-verification",
	}); err != nil {
		t.Fatal(err)
	}

	// When: state eval runs against the observation file
	var stateBuf bytes.Buffer
	app2 := NewApp("t")
	app2.Writer = &stateBuf
	if err := app2.Run([]string{
		"rgd", "state", "eval",
		"--config-dir", cfgDir,
		"--cwd", repo,
		"--observation-file", "obs.json",
	}); err != nil {
		t.Fatal(err)
	}

	// Then: the runtime gate signal is available to state rules
	var out map[string]any
	if err := json.Unmarshal(stateBuf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON: %v raw=%s", err, stateBuf.String())
	}
	if out["kind"] != "resolved" || out["state_id"] != "ready_for_pr" {
		t.Fatalf("unexpected state output: %v", out)
	}
}

func initGitRepoForGateCLI(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGitForGateCLI(t, dir, "init")
	runGitForGateCLI(t, dir, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init")
	runGitForGateCLI(t, dir, "branch", "-M", "main")
	return dir
}

func runGitForGateCLI(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}
