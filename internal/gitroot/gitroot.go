// Package gitroot resolves the git repository root using the git CLI.
package gitroot

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

// Root returns the absolute path of the repository root for the working tree
// containing cwd, using `git rev-parse --show-toplevel`.
func Root(cwd string) (string, error) {
	if cwd == "" {
		return "", fmt.Errorf("gitroot: empty cwd")
	}
	out, err := exec.Command("git", "-C", cwd, "rev-parse", "--show-toplevel").Output()
	if err != nil {
		return "", fmt.Errorf("gitroot: not a git repository (cwd=%s): %w", cwd, err)
	}
	root := strings.TrimSpace(string(out))
	if root == "" {
		return "", fmt.Errorf("gitroot: empty output from git rev-parse")
	}
	return filepath.Clean(root), nil
}
