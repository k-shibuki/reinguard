package main

import (
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"

	"github.com/urfave/cli/v2"
)

func TestRun_version(t *testing.T) {
	t.Parallel()
	// Given/When: main run dispatches version with embedded version id
	// Then: no error
	if err := run([]string{"rgd", "version"}, "testver"); err != nil {
		t.Fatal(err)
	}
}

func TestRun_guardEval_missingObservationFile(t *testing.T) {
	t.Parallel()
	// Given: guard eval with a missing observation file path
	// When: run executes
	err := run([]string{
		"rgd", "guard", "eval",
		"--observation-file", filepath.Join(t.TempDir(), "missing.json"),
		"merge-readiness",
	}, "t")
	// Then: error mentions the file path
	if err == nil || !strings.Contains(err.Error(), "missing.json") {
		t.Fatalf("expected missing observation-file error, got: %v", err)
	}
}

func TestExitStatus(t *testing.T) {
	t.Parallel()

	tests := []struct {
		err          error
		name         string
		wantContains string
		wantCode     int
	}{
		{name: "nil", err: nil, wantCode: 0, wantContains: ""},
		{name: "cli exit", err: cli.Exit("", 2), wantCode: 2, wantContains: ""},
		{name: "generic error", err: errors.New("boom"), wantCode: 1, wantContains: "boom"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			code, message := exitStatus(tt.err)
			if code != tt.wantCode {
				t.Fatalf("expected code %d, got %d", tt.wantCode, code)
			}
			if tt.wantContains == "" {
				if message != "" {
					t.Fatalf("expected empty message, got %q", message)
				}
				return
			}
			if !strings.Contains(message, tt.wantContains) {
				t.Fatalf("expected message containing %q, got %q", tt.wantContains, message)
			}
		})
	}
}

func TestCLIStateEval_failOnNonResolved_exitsTwoWithJSON(t *testing.T) {
	t.Parallel()

	cfgDir := t.TempDir()
	writeTestFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte("schema_version: \"0.7.0\"\ndefault_branch: main\nproviders: []\n"))
	writeTestFile(t, filepath.Join(cfgDir, "control", "states", "rules.yaml"), []byte(`schema_version: "0.7.0"
rules:
  - type: state
    id: a
    priority: 1
    state_id: A
    when: {op: eq, path: git.branch, value: feat}
  - type: state
    id: b
    priority: 1
    state_id: B
    when: {op: eq, path: git.branch, value: feat}
`))
	obsPath := filepath.Join(t.TempDir(), "observation.json")
	writeTestFile(t, obsPath, []byte(`{"schema_version":"0.7.0","signals":{"git":{"branch":"feat"}},"degraded":false}`))

	stdout, stderr, exitCode := runRGDBinary(t, "state", "eval", "--config-dir", cfgDir, "--observation-file", obsPath, "--fail-on-non-resolved")
	if exitCode != 2 {
		t.Fatalf("expected exit 2, got %d stderr=%q", exitCode, stderr)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("invalid JSON: %v raw=%s", err, stdout)
	}
	if out["kind"] != "ambiguous" {
		t.Fatalf("expected ambiguous kind, got %v", out["kind"])
	}
}

func TestCLIRouteSelect_failOnNonResolved_exitsTwoWithJSON(t *testing.T) {
	t.Parallel()

	cfgDir := t.TempDir()
	writeTestFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte("schema_version: \"0.7.0\"\ndefault_branch: main\nproviders: []\n"))
	writeTestFile(t, filepath.Join(cfgDir, "control", "routes", "rules.yaml"), []byte(`schema_version: "0.7.0"
rules:
  - type: route
    id: a
    priority: 1
    route_id: R1
    when: {op: eq, path: git.branch, value: feat}
  - type: route
    id: b
    priority: 1
    route_id: R2
    when: {op: eq, path: git.branch, value: feat}
`))
	obsPath := filepath.Join(t.TempDir(), "observation.json")
	writeTestFile(t, obsPath, []byte(`{"schema_version":"0.7.0","signals":{"git":{"branch":"feat"}},"degraded":false}`))

	stdout, stderr, exitCode := runRGDBinary(t, "route", "select", "--config-dir", cfgDir, "--observation-file", obsPath, "--fail-on-non-resolved")
	if exitCode != 2 {
		t.Fatalf("expected exit 2, got %d stderr=%q", exitCode, stderr)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("invalid JSON: %v raw=%s", err, stdout)
	}
	if out["kind"] != "ambiguous" {
		t.Fatalf("expected ambiguous kind, got %v", out["kind"])
	}
}

func TestCLIContextBuild_failOnNonResolvedState_exitsTwoWithJSON(t *testing.T) {
	t.Parallel()

	cfgDir := t.TempDir()
	writeTestFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte("schema_version: \"0.7.0\"\ndefault_branch: main\nproviders: []\n"))
	writeTestFile(t, filepath.Join(cfgDir, "control", "states", "rules.yaml"), []byte(`schema_version: "0.7.0"
rules:
  - type: state
    id: a
    priority: 1
    state_id: A
    when: {op: eq, path: git.branch, value: feat}
  - type: state
    id: b
    priority: 1
    state_id: B
    when: {op: eq, path: git.branch, value: feat}
`))
	writeTestFile(t, filepath.Join(cfgDir, "control", "routes", "rules.yaml"), []byte(`schema_version: "0.7.0"
rules:
  - type: route
    id: next
    priority: 1
    route_id: next
    when: {op: eq, path: state.kind, value: resolved}
`))
	writeTestFile(t, filepath.Join(cfgDir, "knowledge", "manifest.json"), []byte(`{"schema_version":"0.7.0","entries":[]}`))
	obsPath := filepath.Join(t.TempDir(), "observation.json")
	writeTestFile(t, obsPath, []byte(`{"schema_version":"0.7.0","signals":{"git":{"branch":"feat"}},"degraded":false}`))

	stdout, stderr, exitCode := runRGDBinary(t, "context", "build", "--config-dir", cfgDir, "--observation-file", obsPath, "--fail-on-non-resolved")
	if exitCode != 2 {
		t.Fatalf("expected exit 2, got %d stderr=%q", exitCode, stderr)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("invalid JSON: %v raw=%s", err, stdout)
	}
	state, ok := out["state"].(map[string]any)
	if !ok || state["kind"] != "ambiguous" {
		t.Fatalf("expected ambiguous state, got %v", out["state"])
	}
}

func TestCLIContextBuild_failOnNonResolvedRoute_exitsTwoWithJSON(t *testing.T) {
	t.Parallel()

	cfgDir := t.TempDir()
	writeTestFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte("schema_version: \"0.7.0\"\ndefault_branch: main\nproviders: []\n"))
	writeTestFile(t, filepath.Join(cfgDir, "control", "states", "rules.yaml"), []byte(`schema_version: "0.7.0"
rules:
  - type: state
    id: idle
    priority: 1
    state_id: Idle
    when: {op: eq, path: git.branch, value: main}
`))
	writeTestFile(t, filepath.Join(cfgDir, "control", "routes", "rules.yaml"), []byte("schema_version: \"0.7.0\"\nrules: []\n"))
	writeTestFile(t, filepath.Join(cfgDir, "knowledge", "manifest.json"), []byte(`{"schema_version":"0.7.0","entries":[]}`))
	obsPath := filepath.Join(t.TempDir(), "observation.json")
	writeTestFile(t, obsPath, []byte(`{"schema_version":"0.7.0","signals":{"git":{"branch":"main"}},"degraded":false}`))

	stdout, stderr, exitCode := runRGDBinary(t, "context", "build", "--config-dir", cfgDir, "--observation-file", obsPath, "--fail-on-non-resolved")
	if exitCode != 2 {
		t.Fatalf("expected exit 2, got %d stderr=%q", exitCode, stderr)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("invalid JSON: %v raw=%s", err, stdout)
	}
	routes, ok := out["routes"].([]any)
	if !ok || len(routes) == 0 {
		t.Fatalf("expected routes array, got %v", out["routes"])
	}
	route, ok := routes[0].(map[string]any)
	if !ok || route["kind"] != "degraded" {
		t.Fatalf("expected degraded route, got %v", out["routes"])
	}
}

func TestCLIGateRecord_checksFileFromStdin(t *testing.T) {
	t.Parallel()

	repo := t.TempDir()
	runGit(t, repo, "init")
	runGit(t, repo, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init")
	runGit(t, repo, "branch", "-M", "main")

	cfgDir := t.TempDir()
	stdout, stderr, exitCode := runRGDBinaryWithInput(t, repo, `[
  {"id":"go-test","status":"pass","summary":"go test ./... -race"}
]`, "gate", "record", "--config-dir", cfgDir, "--cwd", repo, "--status", "pass", "--producer-procedure", "implement", "--checks-file", "-", "local-verification")
	if exitCode != 0 {
		t.Fatalf("expected exit 0, got %d stderr=%q", exitCode, stderr)
	}
	var out map[string]any
	if err := json.Unmarshal([]byte(stdout), &out); err != nil {
		t.Fatalf("invalid JSON: %v raw=%s", err, stdout)
	}
	if out["gate_id"] != "local-verification" {
		t.Fatalf("unexpected gate output: %v", out)
	}
}

func runRGDBinary(t *testing.T, args ...string) (stdout string, stderr string, exitCode int) {
	t.Helper()
	return runRGDBinaryWithInput(t, repoRoot(t), "", args...)
}

func runRGDBinaryWithInput(t *testing.T, dir string, stdin string, args ...string) (stdout string, stderr string, exitCode int) {
	t.Helper()

	bin := buildRGDBinary(t)
	cmd := exec.Command(bin, args...)
	cmd.Dir = dir
	cmd.Stdin = strings.NewReader(stdin)
	out, err := cmd.Output()
	if err == nil {
		return string(out), "", 0
	}
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("unexpected command error: %v", err)
	}
	return string(out), string(exitErr.Stderr), exitErr.ExitCode()
}

var (
	buildRGDBinaryOnce sync.Once
	buildRGDBinaryDir  string
	buildRGDBinaryPath string
	buildRGDBinaryErr  error
)

func TestMain(m *testing.M) {
	code := m.Run()
	if buildRGDBinaryDir != "" {
		_ = os.RemoveAll(buildRGDBinaryDir)
	}
	os.Exit(code)
}

func buildRGDBinary(t *testing.T) string {
	t.Helper()

	root := repoRoot(t)
	buildRGDBinaryOnce.Do(func() {
		tmpDir, err := os.MkdirTemp("", "rgd-test-*")
		if err != nil {
			buildRGDBinaryErr = err
			return
		}
		buildRGDBinaryDir = tmpDir
		bin := filepath.Join(tmpDir, "rgd")
		if runtime.GOOS == "windows" {
			bin += ".exe"
		}
		cmd := exec.Command("go", "build", "-o", bin, "./cmd/rgd")
		cmd.Dir = root
		if out, err := cmd.CombinedOutput(); err != nil {
			buildRGDBinaryErr = errors.New(string(out))
			return
		}
		buildRGDBinaryPath = bin
	})
	if buildRGDBinaryErr != nil {
		t.Fatalf("go build rgd: %v", buildRGDBinaryErr)
	}
	return buildRGDBinaryPath
}

func repoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func writeTestFile(t *testing.T, path string, data []byte) {
	t.Helper()

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()

	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v %s", args, err, string(out))
	}
}
