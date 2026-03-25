package githubapi

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// runGHCommand runs the GitHub CLI subprocess. Tests replace it for hermetic runs.
var runGHCommand = runGHCommandImpl

func runGHCommandImpl(ctx context.Context, wd string, args []string) (stdout, stderr []byte, err error) {
	cmd := exec.CommandContext(ctx, "gh", args...)
	if wd != "" {
		cmd.Dir = wd
	}
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return outBuf.Bytes(), errBuf.Bytes(), err
}

// TokenFromGH runs `gh auth token` (ADR-0006).
func TokenFromGH(ctx context.Context, wd string) (string, error) {
	out, stderr, err := runGHCommand(ctx, wd, []string{"auth", "token"})
	if err != nil {
		return "", fmt.Errorf("gh auth token: %w (stderr: %s)", err, strings.TrimSpace(string(stderr)))
	}
	return strings.TrimSpace(string(out)), nil
}

// RepoFromGH runs `gh repo view` for nameWithOwner.
func RepoFromGH(ctx context.Context, wd string) (owner, name string, err error) {
	args := []string{"repo", "view", "--json", "nameWithOwner", "-q", ".nameWithOwner"}
	out, stderr, err := runGHCommand(ctx, wd, args)
	if err != nil {
		return "", "", fmt.Errorf("gh repo view: %w (stderr: %s)", err, strings.TrimSpace(string(stderr)))
	}
	s := strings.TrimSpace(string(out))
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected nameWithOwner %q", s)
	}
	return parts[0], parts[1], nil
}
