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
// (schema_version, default_branch, providers, control/{states,routes,guards}/*.yaml).

const testFixtureReinguardRoot = `schema_version: "0.3.0"
default_branch: main
providers: []
`

const testFixtureReinguardGitOnly = `schema_version: "0.3.0"
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

// Route rule for context build (keys off resolved state.kind, id: r1); pair with testFixtureRulesStateIdle.
const testFixtureControlRoutesNext = `rules:
  - type: route
    id: r1
    priority: 10
    route_id: next
    when:
      op: eq
      path: state.kind
      value: resolved
`

// Two state rules with same priority and overlapping when -> ambiguous with fail-on-non-resolved.
const testFixtureRulesStateAmbiguous = `rules:
  - type: state
    id: a
    priority: 1
    state_id: A
    when: {op: eq, path: x, value: 1}
  - type: state
    id: b
    priority: 1
    state_id: B
    when: {op: eq, path: x, value: 1}
`

// Two route rules with same priority and overlapping when -> ambiguous.
const testFixtureRulesRouteAmbiguous = `rules:
  - type: route
    id: a
    priority: 1
    route_id: R1
    when: {op: eq, path: x, value: 1}
  - type: route
    id: b
    priority: 1
    route_id: R2
    when: {op: eq, path: x, value: 1}
`

const testFixtureRulesEmpty = "rules: []\n"
