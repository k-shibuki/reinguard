package gitroot

import (
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// CurrentBranch returns the checked-out branch name, or ("", true, nil) when HEAD is detached.
// If the working tree is not a valid git checkout, it returns a non-nil error.
func CurrentBranch(ctx context.Context, wd string) (branch string, detached bool, err error) {
	cmd := exec.CommandContext(ctx, "git", "symbolic-ref", "-q", "--short", "HEAD")
	if wd != "" {
		cmd.Dir = wd
	}
	out, err := cmd.Output()
	if err == nil {
		return strings.TrimSpace(string(out)), false, nil
	}
	cmd2 := exec.CommandContext(ctx, "git", "rev-parse", "-q", "--verify", "HEAD")
	if wd != "" {
		cmd2.Dir = wd
	}
	if _, err2 := cmd2.Output(); err2 == nil {
		return "", true, nil
	}
	return "", false, fmt.Errorf("git current branch: %w", err)
}
