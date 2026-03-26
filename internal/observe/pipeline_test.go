package observe

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/k-shibuki/reinguard/internal/config"
)

func TestParseObservationJSON_diagnosticsNonStringFieldsOmitted(t *testing.T) {
	t.Parallel()
	// Given: diagnostics with non-string severity (invalid type coerced to empty)
	data, err := json.Marshal(map[string]any{
		"signals": map[string]any{"x": 1},
		"diagnostics": []any{
			map[string]any{"severity": 123, "message": "hello", "provider": "p", "code": "c"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	// When: ParseObservationJSON runs
	_, diags, _, err := ParseObservationJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	// Then: string fields are empty when JSON types are not strings
	if len(diags) != 1 {
		t.Fatalf("got %d diags", len(diags))
	}
	if diags[0].Severity != "" || diags[0].Message != "hello" {
		t.Fatalf("got severity=%q message=%q", diags[0].Severity, diags[0].Message)
	}
}

func TestParseObservationJSON_stringDiagnostics(t *testing.T) {
	t.Parallel()
	data, err := json.Marshal(map[string]any{
		"signals": map[string]any{"a": true},
		"diagnostics": []any{
			map[string]any{"severity": "warn", "message": "m", "provider": "git", "code": "x"},
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	_, diags, _, err := ParseObservationJSON(data)
	if err != nil {
		t.Fatal(err)
	}
	if len(diags) != 1 || diags[0].Severity != "warn" || diags[0].Message != "m" || diags[0].Provider != "git" || diags[0].Code != "x" {
		t.Fatalf("unexpected diags: %+v", diags)
	}
}

func TestParseObservationJSON_errors(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name       string
		wantSubstr string
		data       []byte
	}{
		{name: "invalid json", wantSubstr: "", data: []byte("{")},
		{name: "missing signals", wantSubstr: "signals", data: []byte(`{"diagnostics":[]}`)},
		{name: "signals not object", wantSubstr: "signals", data: []byte(`{"signals":1}`)},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			_, _, _, err := ParseObservationJSON(tc.data)
			if err == nil {
				t.Fatal("expected error")
			}
			if tc.wantSubstr != "" && !strings.Contains(err.Error(), tc.wantSubstr) {
				t.Fatalf("error %q should mention %q", err.Error(), tc.wantSubstr)
			}
		})
	}
}

func TestLoadSignalsFileOrCollect_readsObservationFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	path := filepath.Join(dir, "obs.json")
	doc := map[string]any{
		"signals":  map[string]any{"git": map[string]any{"k": "v"}},
		"degraded": true,
		"diagnostics": []any{
			map[string]any{"severity": "info", "message": "m", "provider": "p", "code": "c"},
		},
	}
	raw, err := json.Marshal(doc)
	if err != nil {
		t.Fatal(err)
	}
	if writeErr := os.WriteFile(path, raw, 0o600); writeErr != nil {
		t.Fatal(writeErr)
	}
	signals, diags, deg, err := LoadSignalsFileOrCollect(context.Background(), &config.Root{}, LoadSignalsOptions{ObservationPath: path})
	if err != nil {
		t.Fatal(err)
	}
	if !deg {
		t.Fatal("expected degraded true from file")
	}
	gitMap, ok := signals["git"].(map[string]any)
	if !ok || gitMap["k"] != "v" {
		t.Fatalf("signals=%v", signals)
	}
	if len(diags) != 1 || diags[0].Message != "m" {
		t.Fatalf("diags=%+v", diags)
	}
}

func TestLoadSignalsFileOrCollect_readFileError(t *testing.T) {
	t.Parallel()
	missing := filepath.Join(t.TempDir(), "nope.json")
	_, _, _, err := LoadSignalsFileOrCollect(context.Background(), &config.Root{}, LoadSignalsOptions{ObservationPath: missing})
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("want ErrNotExist wrap, got %v", err)
	}
}

func TestLoadSignalsFileOrCollect_collectWhenNoObservationPath(t *testing.T) {
	t.Parallel()
	// Given: no observation file and no enabled providers, live Collect returns empty signals without I/O.
	root := &config.Root{Providers: []config.ProviderSpec{}}
	signals, diags, deg, err := LoadSignalsFileOrCollect(context.Background(), root, LoadSignalsOptions{WorkDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	if deg || len(diags) != 0 {
		t.Fatalf("deg=%v diags=%+v", deg, diags)
	}
	if len(signals) != 0 {
		t.Fatalf("expected empty signals, got %v", signals)
	}
}
