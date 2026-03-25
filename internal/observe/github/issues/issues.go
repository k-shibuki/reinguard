package issues

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/k-shibuki/reinguard/internal/githubapi"
)

type searchResponse struct {
	TotalCount int `json:"total_count"`
}

// Collect returns GitHub Issues signals (open count).
func Collect(ctx context.Context, c *githubapi.Client, owner, repo string) (map[string]any, error) {
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
	return map[string]any{
		"issues": map[string]any{
			"open_count": out.TotalCount,
		},
	}, nil
}
