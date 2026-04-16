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
	// Given: observation JSON with string-typed diagnostic fields
	data, err := json.Marshal(map[string]any{
		"signals": map[string]any{"a": true},
		"diagnostics": []any{
			map[string]any{"severity": "warn", "message": "m", "provider": "git", "code": "x"},
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
	// Then: diagnostics decode with all string fields preserved
	if len(diags) != 1 || diags[0].Severity != "warn" || diags[0].Message != "m" || diags[0].Provider != "git" || diags[0].Code != "x" {
		t.Fatalf("unexpected diags: %+v", diags)
	}
}

func TestParseObservationDocument_meta(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		wantMetaView  string
		wantDegraded  bool
		includeMeta   bool
		metaHasValues bool
	}{
		{
			name:          "meta preserved with degraded",
			wantDegraded:  true,
			wantMetaView:  "summary",
			includeMeta:   true,
			metaHasValues: true,
		},
		{
			name:         "missing meta object",
			wantDegraded: false,
		},
		{
			name:         "empty meta object",
			wantDegraded: true,
			includeMeta:  true,
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			payload := map[string]any{
				"signals":  map[string]any{"a": true},
				"degraded": tc.wantDegraded,
			}
			if tc.includeMeta {
				payload["meta"] = map[string]any{}
			}
			if tc.metaHasValues {
				payload["meta"] = map[string]any{
					"view":             "summary",
					"degraded_sources": []any{"github"},
				}
			}
			data, err := json.Marshal(payload)
			if err != nil {
				t.Fatal(err)
			}
			// When: ParseObservationDocument runs
			doc, err := ParseObservationDocument(data)
			if err != nil {
				t.Fatal(err)
			}
			// Then: optional meta is available to higher-level callers without reparsing
			if doc.Degraded != tc.wantDegraded {
				t.Fatalf("degraded=%v, want=%v", doc.Degraded, tc.wantDegraded)
			}
			if got := stringField(doc.Meta, "view"); got != tc.wantMetaView {
				t.Fatalf("view=%q, want=%q", got, tc.wantMetaView)
			}
			rawSources, ok := doc.Meta["degraded_sources"].([]any)
			if !tc.metaHasValues {
				if ok {
					t.Fatalf("unexpected degraded_sources=%v", rawSources)
				}
				return
			}
			if !ok || len(rawSources) != 1 {
				t.Fatalf("degraded_sources=%v", doc.Meta["degraded_sources"])
			}
			if rawSources[0] != "github" {
				t.Fatalf("degraded_sources[0]=%v, want=%q", rawSources[0], "github")
			}
		})
	}
}

func TestParseObservationJSON_errors(t *testing.T) {
	t.Parallel()
	// Given: malformed observation JSON payloads
	// When: ParseObservationJSON runs
	// Then: non-nil error; message mentions wantSubstr when set
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
	// Given: a valid observation JSON file on disk
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
	// When: LoadSignalsFileOrCollect is called with that path
	signals, diags, deg, err := LoadSignalsFileOrCollect(context.Background(), &config.Root{}, LoadSignalsOptions{ObservationPath: path})
	if err != nil {
		t.Fatal(err)
	}
	// Then: signals, diagnostics, and degraded flag match the file contents
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
	// Given: ObservationPath points to a non-existent file
	missing := filepath.Join(t.TempDir(), "nope.json")
	// When: LoadSignalsFileOrCollect reads it
	_, _, _, err := LoadSignalsFileOrCollect(context.Background(), &config.Root{}, LoadSignalsOptions{ObservationPath: missing})
	// Then: error wraps os.ErrNotExist
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("want ErrNotExist wrap, got %v", err)
	}
}

func TestLoadSignalsFileOrCollect_nilRoot(t *testing.T) {
	t.Parallel()
	// Given: nil config root and live collect path
	// When: LoadSignalsFileOrCollect runs without observation file
	_, _, _, err := LoadSignalsFileOrCollect(context.Background(), nil, LoadSignalsOptions{WorkDir: t.TempDir()})
	// Then: clear error (no panic)
	if err == nil || !strings.Contains(err.Error(), "nil config root") {
		t.Fatalf("got err=%v", err)
	}
}

func TestLoadSignalsFileOrCollect_collectWhenNoObservationPath(t *testing.T) {
	t.Parallel()
	// Given: no observation file and no enabled providers, live Collect returns empty signals without I/O.
	root := &config.Root{Providers: []config.ProviderSpec{}}
	// When: LoadSignalsFileOrCollect runs with empty ObservationPath
	signals, diags, deg, err := LoadSignalsFileOrCollect(context.Background(), root, LoadSignalsOptions{WorkDir: t.TempDir()})
	if err != nil {
		t.Fatal(err)
	}
	// Then: empty signals, no diagnostics, not degraded
	if deg || len(diags) != 0 {
		t.Fatalf("deg=%v diags=%+v", deg, diags)
	}
	if len(signals) != 0 {
		t.Fatalf("expected empty signals, got %v", signals)
	}
}
