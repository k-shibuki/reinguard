package main

import (
	"path/filepath"
	"strings"
	"testing"
)

func TestRun_version(t *testing.T) {
	t.Parallel()
	if err := run([]string{"rgd", "version"}, "testver"); err != nil {
		t.Fatal(err)
	}
}

func TestRun_guardEval_missingObservationFile(t *testing.T) {
	t.Parallel()
	err := run([]string{
		"rgd", "guard", "eval",
		"--observation-file", filepath.Join(t.TempDir(), "missing.json"),
		"merge-readiness",
	}, "t")
	if err == nil || !strings.Contains(err.Error(), "missing.json") {
		t.Fatalf("expected missing observation-file error, got: %v", err)
	}
}
