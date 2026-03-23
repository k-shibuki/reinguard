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
func Export(dir string) (err error) {
	if mkErr := os.MkdirAll(dir, 0o755); mkErr != nil {
		return fmt.Errorf("schema export: mkdir: %w", mkErr)
	}
	src, err := schema.Files().Open(schema.OperationalContextPlaceholder)
	if err != nil {
		return fmt.Errorf("schema export: open embedded: %w", err)
	}
	defer func() {
		if cerr := src.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("schema export: close embedded: %w", cerr)
		}
	}()

	outPath := filepath.Join(dir, schema.OperationalContextPlaceholder)
	dst, err := os.OpenFile(outPath, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0o644)
	if err != nil {
		return fmt.Errorf("schema export: create %s: %w", outPath, err)
	}
	defer func() {
		if cerr := dst.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("schema export: close %s: %w", outPath, cerr)
		}
	}()

	if _, copyErr := io.Copy(dst, src); copyErr != nil {
		return fmt.Errorf("schema export: write %s: %w", outPath, copyErr)
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
