package rgdcli

import (
	"os"
	"path/filepath"
	"testing"
)

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

// Shared fixtures for CLI tests: keep rule shapes aligned with config.Load expectations
// (schema_version, default_branch, providers, rules/*.yaml).

const testFixtureReinguardRoot = `schema_version: "0.2.0"
default_branch: main
providers: []
`

const testFixtureReinguardGitOnly = `schema_version: "0.2.0"
default_branch: main
providers:
  - id: git
    enabled: true
`

// Single state rule: branch main -> Idle (used by state eval / context build smoke tests).
const testFixtureRulesStateIdle = `rules:
  - type: state
    id: idle
    priority: 10
    state_id: Idle
    when:
      op: eq
      path: git.branch
      value: main
`

const testFixtureRulesEmpty = "rules: []\n"
