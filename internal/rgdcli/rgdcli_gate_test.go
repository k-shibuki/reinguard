package rgdcli

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/k-shibuki/reinguard/pkg/schema"
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
		"--producer-procedure", "implement",
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
	if _, ok := recordOut["producer"].(map[string]any); !ok {
		t.Fatalf("missing producer in record output=%v", recordOut)
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
	writeFile(t, filepath.Join(cfgDir, "control", "states", "gates.yaml"), []byte(`schema_version: "0.7.0"
rules:
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
	writeFile(t, filepath.Join(repo, "checks.json"), []byte(`[
  {"id":"go-test","status":"pass","summary":"go test ./... -race"}
]`))

	var recordBuf bytes.Buffer
	app := NewApp("t")
	app.Writer = &recordBuf
	if err := app.Run([]string{
		"rgd", "gate", "record",
		"--config-dir", cfgDir,
		"--cwd", repo,
		"--status", "pass",
		"--producer-procedure", "implement",
		"--checks-file", "checks.json",
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

func TestRunGateRecord_inlineChecks(t *testing.T) {
	t.Parallel()
	repo := initGitRepoForGateCLI(t)
	cfgDir := t.TempDir()

	var buf bytes.Buffer
	app := NewApp("t")
	app.Writer = &buf
	if err := app.Run([]string{
		"rgd", "gate", "record",
		"--config-dir", cfgDir,
		"--cwd", repo,
		"--status", "pass",
		"--producer-procedure", "implement",
		"--check", "go-test:pass:go test ./... -race",
		"--check", "golangci-lint:pass:golangci-lint run",
		"local-verification",
	}); err != nil {
		t.Fatal(err)
	}

	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("json: %v raw=%s", err, buf.String())
	}
	checks, ok := out["checks"].([]any)
	if !ok || len(checks) != 2 {
		t.Fatalf("expected 2 checks, got %v", out)
	}
	got := map[string]string{}
	for _, raw := range checks {
		m, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("check entry must be object: %T", raw)
		}
		id, _ := m["id"].(string)
		status, _ := m["status"].(string)
		summary, _ := m["summary"].(string)
		got[id] = status + "|" + summary
	}
	if got["go-test"] != "pass|go test ./... -race" {
		t.Fatalf("missing/invalid go-test check: %+v", got)
	}
	if got["golangci-lint"] != "pass|golangci-lint run" {
		t.Fatalf("missing/invalid golangci-lint check: %+v", got)
	}
}

func TestRunGateRecord_inlineChecksAndFileChecks(t *testing.T) {
	t.Parallel()
	repo := initGitRepoForGateCLI(t)
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(repo, "checks.json"), []byte(`[
  {"id":"go-test","status":"pass","summary":"go test ./... -race"}
]`))

	var buf bytes.Buffer
	app := NewApp("t")
	app.Writer = &buf
	if err := app.Run([]string{
		"rgd", "gate", "record",
		"--config-dir", cfgDir,
		"--cwd", repo,
		"--status", "pass",
		"--producer-procedure", "implement",
		"--checks-file", "checks.json",
		"--check", "golangci-lint:pass:golangci-lint run",
		"local-verification",
	}); err != nil {
		t.Fatal(err)
	}

	var out map[string]any
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("json: %v raw=%s", err, buf.String())
	}
	checks, ok := out["checks"].([]any)
	if !ok || len(checks) != 2 {
		t.Fatalf("expected 2 checks (1 from file + 1 inline), got %v", out)
	}
	got := map[string]string{}
	for _, raw := range checks {
		m, ok := raw.(map[string]any)
		if !ok {
			t.Fatalf("check entry must be object: %T", raw)
		}
		id, _ := m["id"].(string)
		status, _ := m["status"].(string)
		summary, _ := m["summary"].(string)
		got[id] = status + "|" + summary
	}
	if got["go-test"] != "pass|go test ./... -race" {
		t.Fatalf("missing/invalid file check: %+v", got)
	}
	if got["golangci-lint"] != "pass|golangci-lint run" {
		t.Fatalf("missing/invalid inline check: %+v", got)
	}
}

func TestRunGateRecord_inlineCheckBadFormat(t *testing.T) {
	t.Parallel()
	tests := []string{
		"bad-format",
		":pass:summary",
		"id::summary",
		"id:pass:",
		"id:pass",
	}
	for _, v := range tests {
		v := v
		t.Run(v, func(t *testing.T) {
			t.Parallel()
			repo := initGitRepoForGateCLI(t)
			cfgDir := t.TempDir()

			var buf bytes.Buffer
			app := NewApp("t")
			app.Writer = &buf
			err := app.Run([]string{
				"rgd", "gate", "record",
				"--config-dir", cfgDir,
				"--cwd", repo,
				"--status", "pass",
				"--producer-procedure", "implement",
				"--check", v,
				"local-verification",
			})
			if err == nil {
				t.Fatalf("expected error for malformed --check value %q", v)
			}
			if !strings.Contains(err.Error(), "must be id:status:summary") {
				t.Fatalf("unexpected error for malformed --check value %q: %v", v, err)
			}
		})
	}
}

func TestRunGateRecord_badChecksFile(t *testing.T) {
	t.Parallel()
	// Given: a git repo and a malformed checks file
	repo := initGitRepoForGateCLI(t)
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(repo, "checks.json"), []byte(`{`))

	// When: gate record reads the malformed checks file
	var buf bytes.Buffer
	app := NewApp("t")
	app.Writer = &buf
	err := app.Run([]string{
		"rgd", "gate", "record",
		"--config-dir", cfgDir,
		"--cwd", repo,
		"--status", "pass",
		"--producer-procedure", "implement",
		"--checks-file", "checks.json",
		"local-verification",
	})

	// Then: the JSON parse error is returned
	if err == nil {
		t.Fatal("expected error for malformed checks file")
	}
}

func TestRunGateRecord_resolvesInputGates(t *testing.T) {
	t.Parallel()
	// Given: a git repo with local-verification and local-coderabbit gates recorded
	repo := initGitRepoForGateCLI(t)
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(repo, "verify-checks.json"), []byte(`[
  {"id":"go-test","status":"pass","summary":"go test ./... -race"}
]`))
	writeFile(t, filepath.Join(repo, "cr-checks.json"), []byte(`[
  {"id":"local-coderabbit-cli","status":"pass","summary":"local CodeRabbit completed"}
]`))
	writeFile(t, filepath.Join(repo, "ready-checks.json"), []byte(`[
  {"id":"review-closure","status":"pass","summary":"all local findings classified and closed"}
]`))

	app := NewApp("t")
	app.Writer = &bytes.Buffer{}
	if err := app.Run([]string{
		"rgd", "gate", "record",
		"--config-dir", cfgDir,
		"--cwd", repo,
		"--status", "pass",
		"--producer-procedure", "implement",
		"--checks-file", "verify-checks.json",
		"local-verification",
	}); err != nil {
		t.Fatal(err)
	}
	app2 := NewApp("t")
	app2.Writer = &bytes.Buffer{}
	if err := app2.Run([]string{
		"rgd", "gate", "record",
		"--config-dir", cfgDir,
		"--cwd", repo,
		"--status", "pass",
		"--producer-procedure", "change-inspect",
		"--checks-file", "cr-checks.json",
		"local-coderabbit",
	}); err != nil {
		t.Fatal(err)
	}

	// When: pr-readiness is recorded with both gates as --input-gate dependencies
	var readyBuf bytes.Buffer
	app3 := NewApp("t")
	app3.Writer = &readyBuf
	if err := app3.Run([]string{
		"rgd", "gate", "record",
		"--config-dir", cfgDir,
		"--cwd", repo,
		"--status", "pass",
		"--producer-procedure", "change-inspect",
		"--checks-file", "ready-checks.json",
		"--input-gate", "local-verification",
		"--input-gate", "local-coderabbit",
		"pr-readiness",
	}); err != nil {
		t.Fatal(err)
	}

	// Then: the pr-readiness artifact lists both upstream gates in inputs
	var readyOut map[string]any
	if err := json.Unmarshal(readyBuf.Bytes(), &readyOut); err != nil {
		t.Fatalf("record json: %v raw=%s", err, readyBuf.String())
	}
	inputs, ok := readyOut["inputs"].([]any)
	if !ok || len(inputs) != 2 {
		t.Fatalf("expected two inputs in pr-readiness artifact: %v", readyOut)
	}
	gotIDs := map[string]bool{}
	for _, in := range inputs {
		m, ok := in.(map[string]any)
		if !ok {
			t.Fatalf("input not object: %v", in)
		}
		id, _ := m["gate_id"].(string)
		gotIDs[id] = true
	}
	if !gotIDs["local-verification"] || !gotIDs["local-coderabbit"] {
		t.Fatalf("want local-verification and local-coderabbit in inputs: %+v", inputs)
	}
}

func TestRunGateRecord_prReadinessAllowsOptionalPrePRAIReview(t *testing.T) {
	t.Parallel()
	// Given: reinguard.yaml with pre_pr_ai_review.required=false and local-verification recorded
	repo := initGitRepoForGateCLI(t)
	cfgDir := filepath.Join(repo, ".reinguard")
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(fmt.Sprintf(`schema_version: %q
default_branch: main
workflow:
  runtime_gate_roles:
    pre_pr_ai_review:
      gate_id: local-coderabbit
      required: false
providers: []
`, schema.CurrentSchemaVersion)))
	writeFile(t, filepath.Join(repo, "verify-checks.json"), []byte(`[
  {"id":"go-test","status":"pass","summary":"go test ./... -race"}
]`))
	writeFile(t, filepath.Join(repo, "ready-checks.json"), []byte(`[
  {"id":"review-closure","status":"pass","summary":"all local findings classified and closed"}
]`))

	app := NewApp("t")
	app.Writer = &bytes.Buffer{}
	if err := app.Run([]string{
		"rgd", "gate", "record",
		"--config-dir", cfgDir,
		"--cwd", repo,
		"--status", "pass",
		"--producer-procedure", "implement",
		"--checks-file", "verify-checks.json",
		"local-verification",
	}); err != nil {
		t.Fatal(err)
	}

	// When: pr-readiness is recorded with only local-verification as input (optional AI review omitted)
	var readyBuf bytes.Buffer
	app2 := NewApp("t")
	app2.Writer = &readyBuf
	if err := app2.Run([]string{
		"rgd", "gate", "record",
		"--config-dir", cfgDir,
		"--cwd", repo,
		"--status", "pass",
		"--producer-procedure", "change-inspect",
		"--checks-file", "ready-checks.json",
		"--input-gate", "local-verification",
		"pr-readiness",
	}); err != nil {
		t.Fatal(err)
	}

	// Then: inputs contains only local-verification
	var readyOut map[string]any
	if err := json.Unmarshal(readyBuf.Bytes(), &readyOut); err != nil {
		t.Fatalf("record json: %v raw=%s", err, readyBuf.String())
	}
	inputs, ok := readyOut["inputs"].([]any)
	if !ok || len(inputs) != 1 {
		t.Fatalf("expected one input in optional AI review config: %v", readyOut)
	}
	m0, ok := inputs[0].(map[string]any)
	if !ok || m0["gate_id"] != "local-verification" {
		t.Fatalf("expected sole input gate_id local-verification: %v", inputs[0])
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
