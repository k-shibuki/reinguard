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

// checkRunJSON is one element of check_runs from GET .../commits/{sha}/check-runs.
type checkRunJSON struct {
	Conclusion *string `json:"conclusion"`
	Name       string  `json:"name"`
	Status     string  `json:"status"`
}

// checkRunsAPIResponse is the JSON shape of GET /repos/{owner}/{repo}/commits/{sha}/check-runs.
type checkRunsAPIResponse struct {
	CheckRuns  []checkRunJSON `json:"check_runs"`
	TotalCount int            `json:"total_count"`
}

// Supported CI facet views.
const (
	ViewSummary = "summary"
	ViewFull    = "full"
)

// Collect returns a coarse CI rollup for the observed head SHA.
// If headSHAOverride is non-empty after trimming, it is used directly; otherwise the
// SHA is determined via git rev-parse HEAD in workDir.
// owner and repo identify the repository for GET .../commits/{sha}/status. For a pull
// request from a fork, CI statuses are posted to the head repository; pass the head
// owner and name from the pull request (not the base repo).
func Collect(ctx context.Context, c *githubapi.Client, owner, repo, workDir, headSHAOverride, view string) (map[string]any, []string, error) {
	if c == nil {
		return nil, nil, fmt.Errorf("nil client")
	}
	var warnings []string
	view = strings.ToLower(strings.TrimSpace(view))
	if view == "" {
		view = ViewFull
	}
	if view != ViewSummary && view != ViewFull {
		warnings = append(warnings, fmt.Sprintf("github ci: unknown view %q, defaulting to %s", view, ViewFull))
		view = ViewFull
	}
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
	if err = c.GetJSON(ctx, u, &st); err != nil {
		return nil, warnings, err
	}
	status := strings.ToLower(st.State)
	if status == "" {
		status = "unknown"
	}
	ciMap := map[string]any{
		"ci_status": status,
		"head_sha":  sha,
	}
	if view == ViewFull {
		checkRuns, warnsCR, err := fetchCheckRuns(ctx, c, owner, repo, sha)
		warnings = append(warnings, warnsCR...)
		if err != nil {
			warnings = append(warnings, err.Error())
			ciMap["check_runs"] = []any{}
		} else {
			ciMap["check_runs"] = checkRuns
		}
	}
	return map[string]any{
		"ci": ciMap,
	}, warnings, nil
}

func fetchCheckRuns(ctx context.Context, c *githubapi.Client, owner, repo, sha string) ([]any, []string, error) {
	if c == nil {
		return nil, nil, fmt.Errorf("nil client")
	}
	u := fmt.Sprintf("%s/repos/%s/%s/commits/%s/check-runs?per_page=100",
		c.APIBase(),
		url.PathEscape(owner),
		url.PathEscape(repo),
		url.PathEscape(sha),
	)
	var resp checkRunsAPIResponse
	if err := c.GetJSON(ctx, u, &resp); err != nil {
		return nil, nil, err
	}
	var warns []string
	if resp.TotalCount > len(resp.CheckRuns) {
		warns = append(warns, fmt.Sprintf("github ci: check-runs response truncated (%d of %d)", len(resp.CheckRuns), resp.TotalCount))
	}
	out := make([]any, 0, len(resp.CheckRuns))
	for _, cr := range resp.CheckRuns {
		entry := map[string]any{
			"name":   cr.Name,
			"status": strings.ToLower(strings.TrimSpace(cr.Status)),
		}
		if cr.Conclusion != nil {
			entry["conclusion"] = strings.ToLower(strings.TrimSpace(*cr.Conclusion))
		} else {
			entry["conclusion"] = nil
		}
		out = append(out, entry)
	}
	return out, warns, nil
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
