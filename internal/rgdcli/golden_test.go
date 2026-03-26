package rgdcli

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestGolden_stateEval(t *testing.T) {
	t.Parallel()
	// Given: fixture config dir and golden observation/want JSON
	cfgDir := goldenSetupConfigDir(t, testFixtureRulesStateIdle, testFixtureControlRoutesNext)
	dir := goldenCaseDir(t, "state_eval")
	obs := filepath.Join(dir, "observation.json")
	wantRaw := readFile(t, filepath.Join(dir, "want.json"))

	var buf bytes.Buffer
	app := NewApp("test")
	app.Writer = &buf
	app.ErrWriter = &bytes.Buffer{}
	// When: state eval runs against golden observation
	err := app.Run([]string{"rgd", "state", "eval", "--config-dir", cfgDir, "--observation-file", obs})
	if err != nil {
		t.Fatal(err)
	}
	// Then: stdout JSON matches want.json canonically
	assertCanonicalJSONEqual(t, buf.Bytes(), wantRaw)
}

func TestGolden_contextBuild(t *testing.T) {
	t.Parallel()
	// Given: fixture config dir and golden observation/want JSON
	cfgDir := goldenSetupConfigDir(t, testFixtureRulesStateIdle, testFixtureControlRoutesNext)
	dir := goldenCaseDir(t, "context_build")
	obs := filepath.Join(dir, "observation.json")
	wantRaw := readFile(t, filepath.Join(dir, "want.json"))

	var buf bytes.Buffer
	app := NewApp("test")
	app.Writer = &buf
	app.ErrWriter = &bytes.Buffer{}
	// When: context build runs against golden observation
	err := app.Run([]string{"rgd", "context", "build", "--config-dir", cfgDir, "--observation-file", obs})
	if err != nil {
		t.Fatal(err)
	}
	// Then: stdout JSON matches want.json canonically
	assertCanonicalJSONEqual(t, buf.Bytes(), wantRaw)
}

func goldenCaseDir(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	return filepath.Join(filepath.Dir(file), "testdata", "golden", name)
}

func goldenSetupConfigDir(t *testing.T, stateYAML, routeYAML string) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	if stateYAML != "" {
		writeFile(t, filepath.Join(root, "control", "states", "default.yaml"), []byte(stateYAML))
	}
	if routeYAML != "" {
		writeFile(t, filepath.Join(root, "control", "routes", "default.yaml"), []byte(routeYAML))
	}
	if err := os.Mkdir(filepath.Join(root, "knowledge"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "knowledge", "doc.md"), []byte(`---
id: doc1
description: fixture knowledge for golden tests
triggers:
  - fixture
---

# Doc
`))
	writeFile(t, filepath.Join(root, "knowledge", "manifest.json"), []byte(`{
  "schema_version": "0.3.0",
  "entries": [{
    "id": "doc1",
    "path": "knowledge/doc.md",
    "description": "fixture knowledge for golden tests",
    "triggers": ["fixture"]
  }]
}`))
	return root
}

func readFile(t *testing.T, p string) []byte {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func assertCanonicalJSONEqual(t *testing.T, gotJSON, wantJSON []byte) {
	t.Helper()
	var got, want any
	if err := json.Unmarshal(gotJSON, &got); err != nil {
		t.Fatalf("got: %v", err)
	}
	if err := json.Unmarshal(wantJSON, &want); err != nil {
		t.Fatalf("want: %v", err)
	}
	gb, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	wb, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	if string(gb) != string(wb) {
		t.Fatalf("JSON mismatch\ngot:  %s\nwant: %s", gb, wb)
	}
}
