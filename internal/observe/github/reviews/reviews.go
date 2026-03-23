package reviews

import (
	"context"
	"fmt"

	"github.com/k-shibuki/reinguard/internal/githubapi"
)

// Collect returns minimal review-comment signals for an open PR (Phase 1 subset).
func Collect(ctx context.Context, c *githubapi.Client, owner, repo string, prNumber int) (map[string]any, error) {
	if c == nil {
		return nil, fmt.Errorf("nil client")
	}
	unresolved := 0
	incomplete := false
	if prNumber > 0 {
		u := fmt.Sprintf(
			"%s/repos/%s/%s/pulls/%d/comments?per_page=100",
			c.APIBase(), owner, repo, prNumber,
		)
		var comments []any
		if err := c.GetJSON(ctx, u, &comments); err != nil {
			return nil, err
		}
		unresolved = len(comments)
	}
	return map[string]any{
		"reviews": map[string]any{
			"review_threads_unresolved": unresolved,
			"pagination_incomplete":     incomplete,
		},
	}, nil
}
