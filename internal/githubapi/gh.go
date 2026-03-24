package githubapi

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// TokenFromGH runs `gh auth token` (ADR-0006).
func TokenFromGH(ctx context.Context, wd string) (string, error) {
	cmd := exec.CommandContext(ctx, "gh", "auth", "token")
	if wd != "" {
		cmd.Dir = wd
	}
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gh auth token: %w (stderr: %s)", err, strings.TrimSpace(errBuf.String()))
	}
	return strings.TrimSpace(outBuf.String()), nil
}

// RepoFromGH runs `gh repo view` for nameWithOwner.
func RepoFromGH(ctx context.Context, wd string) (owner, name string, err error) {
	cmd := exec.CommandContext(ctx, "gh", "repo", "view", "--json", "nameWithOwner", "-q", ".nameWithOwner")
	if wd != "" {
		cmd.Dir = wd
	}
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	if err := cmd.Run(); err != nil {
		return "", "", fmt.Errorf("gh repo view: %w (stderr: %s)", err, strings.TrimSpace(errBuf.String()))
	}
	s := strings.TrimSpace(outBuf.String())
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected nameWithOwner %q", s)
	}
	return parts[0], parts[1], nil
}
