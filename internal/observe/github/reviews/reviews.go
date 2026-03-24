package reviews

import (
	"context"
	"fmt"

	"github.com/k-shibuki/reinguard/internal/githubapi"
)

const reviewThreadsQuery = `
query ReviewThreads($owner: String!, $name: String!, $n: Int!) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $n) {
      reviewThreads(first: 100) {
        pageInfo {
          hasNextPage
        }
        nodes {
          isResolved
        }
      }
    }
  }
}
`

type gqlReviewThreadsResponse struct {
	Data struct {
		Repository *struct {
			PullRequest *struct {
				ReviewThreads struct {
					Nodes []struct {
						IsResolved bool `json:"isResolved"`
					} `json:"nodes"`
					PageInfo struct {
						HasNextPage bool `json:"hasNextPage"`
					} `json:"pageInfo"`
				} `json:"reviewThreads"`
			} `json:"pullRequest"`
		} `json:"repository"`
	} `json:"data"`
	Errors []struct {
		Message string `json:"message"`
	} `json:"errors"`
}

// Collect returns review-thread signals for an open PR (unresolved thread count via GraphQL).
func Collect(ctx context.Context, c *githubapi.Client, owner, repo string, prNumber int) (map[string]any, error) {
	if c == nil {
		return nil, fmt.Errorf("nil client")
	}
	unresolved := 0
	incomplete := false
	if prNumber > 0 {
		var resp gqlReviewThreadsResponse
		err := c.PostGraphQL(ctx, reviewThreadsQuery, map[string]any{
			"owner": owner,
			"name":  repo,
			"n":     prNumber,
		}, &resp)
		if err != nil {
			return nil, err
		}
		if len(resp.Errors) > 0 {
			return nil, fmt.Errorf("graphql: %s", resp.Errors[0].Message)
		}
		if resp.Data.Repository != nil && resp.Data.Repository.PullRequest != nil {
			rt := resp.Data.Repository.PullRequest.ReviewThreads
			for _, n := range rt.Nodes {
				if !n.IsResolved {
					unresolved++
				}
			}
			incomplete = rt.PageInfo.HasNextPage || len(rt.Nodes) >= 100
		}
	}
	return map[string]any{
		"reviews": map[string]any{
			"review_threads_unresolved": unresolved,
			"pagination_incomplete":     incomplete,
		},
	}, nil
}
