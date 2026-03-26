// Package configdir resolves the .reinguard configuration directory.
package configdir

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/k-shibuki/reinguard/internal/gitroot"
)

// Resolve returns the absolute path to the directory containing reinguard
// configuration files. If explicit is non-empty, it is treated as a path to
// that directory (absolute or relative to cwd). Otherwise, the git repository
// root is discovered and ".reinguard" under that root is used.
func Resolve(cwd, explicit string) (string, error) {
	if cwd == "" {
		return "", fmt.Errorf("configdir: empty cwd")
	}
	if explicit != "" {
		if filepath.IsAbs(explicit) {
			return filepath.Clean(explicit), nil
		}
		abs, err := filepath.Abs(filepath.Join(cwd, explicit))
		if err != nil {
			return "", err
		}
		return filepath.Clean(abs), nil
	}
	root, err := gitroot.Root(cwd)
	if err != nil {
		return "", fmt.Errorf(
			"configdir: %w (hint: pass --config-dir to point at your config directory)",
			err,
		)
	}
	return filepath.Join(root, ".reinguard"), nil
}

// WorkingDir returns an absolute working directory path.
func WorkingDir() (string, error) {
	return os.Getwd()
}

// RepoRoot resolves the repository root for repo-relative manifest paths (ADR-0010).
// When the config directory basename is ".reinguard", its parent is the repo root.
// Otherwise the config directory is treated as the repo root (flat test layouts).
func RepoRoot(cfgDir string) string {
	cfgDir = filepath.Clean(cfgDir)
	if filepath.Base(cfgDir) == ".reinguard" {
		return filepath.Clean(filepath.Join(cfgDir, ".."))
	}
	return cfgDir
}
