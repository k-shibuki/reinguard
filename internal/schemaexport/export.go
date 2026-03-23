// Package schemaexport writes embedded JSON Schemas to disk (MVP).
package schemaexport

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/k-shibuki/reinguard/pkg/schema"
)

// Export writes the embedded operational context placeholder schema into dir.
func Export(dir string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("schema export: mkdir: %w", err)
	}
	src, err := schema.Files().Open(schema.OperationalContextPlaceholder)
	if err != nil {
		return fmt.Errorf("schema export: open embedded: %w", err)
	}
	defer src.Close()

	outPath := filepath.Join(dir, schema.OperationalContextPlaceholder)
	dst, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("schema export: create %s: %w", outPath, err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("schema export: write %s: %w", outPath, err)
	}
	return nil
}

// ListEmbedded returns embedded schema entry names (for tests / future CLI).
func ListEmbedded() ([]string, error) {
	var names []string
	err := fs.WalkDir(schema.Files(), ".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		names = append(names, path)
		return nil
	})
	return names, err
}
