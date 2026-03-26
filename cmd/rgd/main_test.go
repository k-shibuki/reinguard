package main

import (
	"path/filepath"
	"strings"
	"testing"
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
