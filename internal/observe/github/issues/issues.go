// Package issues implements open-issue count signals via GitHub search for the observe GitHub
// provider (ADR-0006). Collect returns an issues subtree or an error on invalid input or API failure.
package issues

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/k-shibuki/reinguard/internal/githubapi"
)

// searchResponse is the total_count field from GitHub issue search API results.
type searchResponse struct {
	TotalCount int `json:"total_count"`
}

// Collect returns GitHub Issues signals (open_count; optional selected_issues when issueNumbers is non-empty).
func Collect(ctx context.Context, c *githubapi.Client, owner, repo string, issueNumbers []int) (map[string]any, error) {
	if c == nil {
		return nil, fmt.Errorf("nil client")
	}
	owner = strings.TrimSpace(owner)
	repo = strings.TrimSpace(repo)
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("issues: owner and repo must be non-empty")
	}
	q := fmt.Sprintf("repo:%s/%s is:issue is:open", owner, repo)
	u := c.APIBase() + "/search/issues?q=" + url.QueryEscape(q)
	var out searchResponse
	if err := c.GetJSON(ctx, u, &out); err != nil {
		return nil, fmt.Errorf("fetch open issues for %s/%s: %w", owner, repo, err)
	}
	inner := map[string]any{
		"open_count": out.TotalCount,
	}
	nums := dedupeIssueNumbers(issueNumbers)
	if len(nums) > 0 {
		selected, err := CollectSelected(ctx, c, owner, repo, nums)
		if err != nil {
			return nil, err
		}
		inner["selected_issues"] = selected
	}
	return map[string]any{
		"issues": inner,
	}, nil
}

func dedupeIssueNumbers(in []int) []int {
	if len(in) == 0 {
		return nil
	}
	seen := make(map[int]struct{}, len(in))
	out := make([]int, 0, len(in))
	for _, n := range in {
		if _, ok := seen[n]; ok {
			continue
		}
		seen[n] = struct{}{}
		out = append(out, n)
	}
	return out
}
