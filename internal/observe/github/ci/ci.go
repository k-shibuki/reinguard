package ci

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/k-shibuki/reinguard/internal/githubapi"
)

type combinedStatus struct {
	State string `json:"state"`
}

// Collect returns a coarse CI rollup for HEAD of the current branch.
func Collect(ctx context.Context, c *githubapi.Client, owner, repo, workDir string) (map[string]any, []string, error) {
	if c == nil {
		return nil, nil, fmt.Errorf("nil client")
	}
	var warnings []string
	sha, err := headSHA(ctx, workDir)
	if err != nil {
		warnings = append(warnings, err.Error())
		return map[string]any{
			"ci": map[string]any{
				"ci_status": "unknown",
				"head_sha":  "",
			},
		}, warnings, nil
	}
	u := fmt.Sprintf("%s/repos/%s/%s/commits/%s/status", c.APIBase(), owner, repo, sha)
	var st combinedStatus
	if err := c.GetJSON(ctx, u, &st); err != nil {
		return nil, warnings, err
	}
	status := strings.ToLower(st.State)
	if status == "" {
		status = "unknown"
	}
	return map[string]any{
		"ci": map[string]any{
			"ci_status": status,
			"head_sha":  sha,
		},
	}, warnings, nil
}

func headSHA(ctx context.Context, wd string) (string, error) {
	var buf bytes.Buffer
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "HEAD")
	if wd != "" {
		cmd.Dir = wd
	}
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git rev-parse: %w: %s", err, strings.TrimSpace(buf.String()))
	}
	return strings.TrimSpace(buf.String()), nil
}
