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
	cfgDir := goldenSetupConfigDir(t, testFixtureRulesContextBuild)
	dir := goldenCaseDir(t, "state_eval")
	obs := filepath.Join(dir, "observation.json")
	wantRaw := readFile(t, filepath.Join(dir, "want.json"))

	var buf bytes.Buffer
	app := NewApp("test")
	app.Writer = &buf
	app.ErrWriter = &bytes.Buffer{}
	err := app.Run([]string{"rgd", "state", "eval", "--config-dir", cfgDir, "--observation-file", obs})
	if err != nil {
		t.Fatal(err)
	}
	assertCanonicalJSONEqual(t, buf.Bytes(), wantRaw)
}

func TestGolden_contextBuild(t *testing.T) {
	t.Parallel()
	cfgDir := goldenSetupConfigDir(t, testFixtureRulesContextBuild)
	dir := goldenCaseDir(t, "context_build")
	obs := filepath.Join(dir, "observation.json")
	wantRaw := readFile(t, filepath.Join(dir, "want.json"))

	var buf bytes.Buffer
	app := NewApp("test")
	app.Writer = &buf
	app.ErrWriter = &bytes.Buffer{}
	err := app.Run([]string{"rgd", "context", "build", "--config-dir", cfgDir, "--observation-file", obs})
	if err != nil {
		t.Fatal(err)
	}
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

func goldenSetupConfigDir(t *testing.T, rulesYAML string) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	if err := os.Mkdir(filepath.Join(root, "rules"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "rules", "default.yaml"), []byte(rulesYAML))
	if err := os.Mkdir(filepath.Join(root, "knowledge"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "knowledge", "manifest.json"), []byte(`{
  "schema_version": "0.2.0",
  "entries": [{"id": "doc1", "path": ".reinguard/README.md"}]
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
