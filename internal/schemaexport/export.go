// Package schemaexport writes embedded JSON Schemas from pkg/schema to a directory on
// disk (used by the rgd schema export command and maintainers who want files beside the binary).
package schemaexport

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/k-shibuki/reinguard/pkg/schema"
)

// Export writes all embedded JSON schemas into dir.
func Export(dir string) (err error) {
	if mkErr := os.MkdirAll(dir, 0o755); mkErr != nil {
		return fmt.Errorf("schema export: mkdir: %w", mkErr)
	}
	return fs.WalkDir(schema.Files(), ".", func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		return copyEmbeddedFile(dir, path)
	})
}

func copyEmbeddedFile(destDir, name string) (err error) {
	src, err := schema.Files().Open(name)
	if err != nil {
		return fmt.Errorf("schema export: open embedded %s: %w", name, err)
	}
	defer func() {
		if cerr := src.Close(); cerr != nil && err == nil {
			err = fmt.Errorf("schema export: close embedded %s: %w", name, cerr)
		}
	}()

	outPath := filepath.Join(destDir, filepath.Base(name))
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

// ListEmbedded returns embedded schema entry names (for tests / CLI).
func ListEmbedded() ([]string, error) {
	return schema.ListEmbedded()
}
