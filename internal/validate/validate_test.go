package validate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDir_ok(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "reinguard.yaml"), []byte("schema_version: \"0.1.0\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := Dir(dir); err != nil {
		t.Fatal(err)
	}
}

func TestDir_missingFile(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	if err := Dir(dir); err == nil {
		t.Fatal("expected error")
	}
}
