// Package reviews implements PR review thread signals for the observe GitHub provider
// (ADR-0006, ADR-0012). Collect uses GraphQL reviewThreads because REST review comments
// do not expose per-thread isResolved (see .reinguard/knowledge/review--github-thread-api.md).
package reviews

import (
	"context"
	"fmt"

	"github.com/k-shibuki/reinguard/internal/githubapi"
)

// reviewThreadsQuery loads review thread nodes with isResolved for pagination.
const reviewThreadsQuery = `query ReviewThreads($owner: String!, $name: String!, $number: Int!, $cursor: String) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      reviewThreads(first: 100, after: $cursor) {
        pageInfo {
          hasNextPage
          endCursor
        }
        nodes {
          isResolved
        }
      }
    }
  }
}`

// maxReviewThreadPages caps pagination to avoid unbounded work; if more pages exist,
// pagination_incomplete is true and counts include only fetched pages.
const maxReviewThreadPages = 500

// reviewThreadsResponse mirrors GitHub GraphQL JSON layout for json.Unmarshal.
//
//nolint:govet // fieldalignment: keep field order aligned with API response shape
type reviewThreadsResponse struct {
	Repository *struct {
		PullRequest *struct {
			ReviewThreads *struct {
				Nodes []struct {
					IsResolved bool `json:"isResolved"`
				} `json:"nodes"`
				PageInfo struct {
					HasNextPage bool   `json:"hasNextPage"`
					EndCursor   string `json:"endCursor"`
				} `json:"pageInfo"`
			} `json:"reviewThreads"`
		} `json:"pullRequest"`
	} `json:"repository"`
}

// Collect returns review thread counts for an open PR using GraphQL reviewThreads.
func Collect(ctx context.Context, c *githubapi.Client, owner, repo string, prNumber int) (map[string]any, error) {
	if c == nil {
		return nil, fmt.Errorf("nil client")
	}
	total := 0
	unresolved := 0
	incomplete := false
	if prNumber <= 0 {
		return map[string]any{
			"reviews": map[string]any{
				"review_threads_total":      0,
				"review_threads_unresolved": 0,
				"pagination_incomplete":     false,
			},
		}, nil
	}

	var cursor any
	for page := 0; page < maxReviewThreadPages; page++ {
		vars := map[string]any{
			"owner":  owner,
			"name":   repo,
			"number": prNumber,
			"cursor": cursor,
		}
		var data reviewThreadsResponse
		if err := c.PostGraphQL(ctx, reviewThreadsQuery, vars, &data); err != nil {
			return nil, err
		}
		if data.Repository == nil || data.Repository.PullRequest == nil {
			break
		}
		rt := data.Repository.PullRequest.ReviewThreads
		if rt == nil {
			break
		}
		for _, n := range rt.Nodes {
			total++
			if !n.IsResolved {
				unresolved++
			}
		}
		if !rt.PageInfo.HasNextPage {
			break
		}
		if page == maxReviewThreadPages-1 {
			incomplete = true
			break
		}
		cursor = rt.PageInfo.EndCursor
		if cursor == "" {
			incomplete = true
			break
		}
	}

	return map[string]any{
		"reviews": map[string]any{
			"review_threads_total":      total,
			"review_threads_unresolved": unresolved,
			"pagination_incomplete":     incomplete,
		},
	}, nil
}
