package main

import (
	"path/filepath"
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
		"any-guard",
	}, "t")
	if err == nil {
		t.Fatal("expected error")
	}
}
