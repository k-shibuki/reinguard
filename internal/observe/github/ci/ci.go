// Package ci implements the GitHub commit combined status facet used by the observe GitHub
// provider (ADR-0006 REST observation, ADR-0009 provider composition). Collect returns a
// ci subtree and optional warnings; errors from the API are returned as Go errors.
package ci

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"strings"

	"github.com/k-shibuki/reinguard/internal/githubapi"
)

// combinedStatus is the JSON shape of GET /repos/{owner}/{repo}/commits/{sha}/status.
type combinedStatus struct {
	State string `json:"state"`
}

// Collect returns a coarse CI rollup for the observed head SHA.
// If headSHAOverride is non-empty after trimming, it is used directly; otherwise the
// SHA is determined via git rev-parse HEAD in workDir.
func Collect(ctx context.Context, c *githubapi.Client, owner, repo, workDir, headSHAOverride string) (map[string]any, []string, error) {
	if c == nil {
		return nil, nil, fmt.Errorf("nil client")
	}
	var warnings []string
	sha := strings.TrimSpace(headSHAOverride)
	var err error
	if sha == "" {
		sha, err = headSHA(ctx, workDir)
	}
	if err != nil {
		warnings = append(warnings, err.Error())
		return map[string]any{
			"ci": map[string]any{
				"ci_status": "unknown",
				"head_sha":  "",
			},
		}, warnings, nil
	}
	u := fmt.Sprintf("%s/repos/%s/%s/commits/%s/status",
		c.APIBase(),
		url.PathEscape(owner),
		url.PathEscape(repo),
		url.PathEscape(sha),
	)
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

// headSHA returns the current commit SHA from git rev-parse HEAD in wd.
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
