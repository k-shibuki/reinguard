package rgdcli

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunGuardEval_unknownGuard(t *testing.T) {
	t.Parallel()
	// Given: a valid observation file with empty signals
	var buf bytes.Buffer
	app := NewApp("t")
	app.Writer = &buf
	tmp := t.TempDir()
	p := filepath.Join(tmp, "o.json")
	writeFile(t, p, []byte(`{"signals":{}}`))

	// When: guard eval is invoked with a non-existent guard name
	err := app.Run([]string{"rgd", "guard", "eval", "--observation-file", p, "not-a-guard"})

	// Then: an "unknown guard" error is returned
	if err == nil || !strings.Contains(err.Error(), "unknown guard") {
		t.Fatalf("%v", err)
	}
}

func TestRunGuardEval_badJSON(t *testing.T) {
	t.Parallel()
	// Given: an observation file containing invalid JSON
	tmp := t.TempDir()
	p := filepath.Join(tmp, "bad.json")
	writeFile(t, p, []byte(`{`))

	// When: guard eval is invoked against the malformed file
	var buf bytes.Buffer
	app := NewApp("t")
	app.Writer = &buf
	err := app.Run([]string{"rgd", "guard", "eval", "--observation-file", p, "merge-readiness"})

	// Then: a JSON parse error is returned (not some other failure)
	if err == nil {
		t.Fatal("expected json error")
	}
	var syn *json.SyntaxError
	if !errors.As(err, &syn) {
		t.Fatalf("expected JSON syntax error, got %T: %v", err, err)
	}
}

func TestRunGuardEval_ok(t *testing.T) {
	t.Parallel()
	// Given: an observation file with all merge-readiness signals satisfied
	tmp := t.TempDir()
	p := filepath.Join(tmp, "o.json")
	writeFile(t, p, []byte(`{
	  "signals": {
	    "git": {"working_tree_clean": true},
	    "github": {
	      "ci": {"ci_status": "success"},
	      "reviews": {"review_threads_unresolved": 0}
	    }
	  }
	}`))

	// When: guard eval is invoked for merge-readiness
	var buf bytes.Buffer
	app := NewApp("t")
	app.Writer = &buf
	if err := app.Run([]string{"rgd", "guard", "eval", "--observation-file", p, "merge-readiness"}); err != nil {
		t.Fatal(err)
	}

	// Then: decoded output has ok=true (not tied to pretty-print whitespace)
	var out struct {
		OK bool `json:"ok"`
	}
	if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
		t.Fatalf("invalid JSON output: %v; raw=%s", err, buf.String())
	}
	if !out.OK {
		t.Fatalf("expected ok=true, got %+v", buf.String())
	}
}
