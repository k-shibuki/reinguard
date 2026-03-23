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
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("gh auth token: %w: %s", err, strings.TrimSpace(buf.String()))
	}
	return strings.TrimSpace(buf.String()), nil
}

// RepoFromGH runs `gh repo view` for nameWithOwner.
func RepoFromGH(ctx context.Context, wd string) (owner, name string, err error) {
	cmd := exec.CommandContext(ctx, "gh", "repo", "view", "--json", "nameWithOwner", "-q", ".nameWithOwner")
	if wd != "" {
		cmd.Dir = wd
	}
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return "", "", fmt.Errorf("gh repo view: %w: %s", err, strings.TrimSpace(buf.String()))
	}
	s := strings.TrimSpace(buf.String())
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected nameWithOwner %q", s)
	}
	return parts[0], parts[1], nil
}
