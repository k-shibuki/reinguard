package rgdcli

import (
	"bytes"
	"path/filepath"
	"strings"
	"testing"
)

func TestRunGuardEval_unknownGuard(t *testing.T) {
	t.Parallel()
	var buf bytes.Buffer
	app := NewApp("t")
	app.Writer = &buf
	tmp := t.TempDir()
	p := filepath.Join(tmp, "o.json")
	writeFile(t, p, []byte(`{"signals":{}}`))
	err := app.Run([]string{"rgd", "guard", "eval", "--observation-file", p, "not-a-guard"})
	if err == nil || !strings.Contains(err.Error(), "unknown guard") {
		t.Fatalf("%v", err)
	}
}

func TestRunGuardEval_badJSON(t *testing.T) {
	t.Parallel()
	tmp := t.TempDir()
	p := filepath.Join(tmp, "bad.json")
	writeFile(t, p, []byte(`{`))
	var buf bytes.Buffer
	app := NewApp("t")
	app.Writer = &buf
	err := app.Run([]string{"rgd", "guard", "eval", "--observation-file", p, "merge-readiness"})
	if err == nil {
		t.Fatal("expected json error")
	}
}

func TestRunGuardEval_ok(t *testing.T) {
	t.Parallel()
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
	var buf bytes.Buffer
	app := NewApp("t")
	app.Writer = &buf
	if err := app.Run([]string{"rgd", "guard", "eval", "--observation-file", p, "merge-readiness"}); err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(buf.Bytes(), []byte(`"ok": true`)) {
		t.Fatalf("%s", buf.String())
	}
}
