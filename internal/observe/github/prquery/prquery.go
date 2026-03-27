// Package prquery runs a unified GitHub GraphQL query for the current PR: detail, review
// threads, latest review decisions, linked issues, and PR comments for tracked reviewers
// (ADR-0006, ADR-0012).
package prquery

import (
	"context"
	"fmt"
	"strings"

	"github.com/k-shibuki/reinguard/internal/githubapi"
)

// TrackedReviewer is one github provider option entry for bot / reviewer status observation.
type TrackedReviewer struct {
	Login  string
	Enrich []string
}

const prContextQuery = `query PRContext($owner: String!, $name: String!, $number: Int!, $cursor: String, $includeDetail: Boolean!) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      state @include(if: $includeDetail)
      isDraft @include(if: $includeDetail)
      title @include(if: $includeDetail)
      mergeable @include(if: $includeDetail)
      mergeStateStatus @include(if: $includeDetail)
      baseRefName @include(if: $includeDetail)
      headRefOid @include(if: $includeDetail)
      labels(first: 20) @include(if: $includeDetail) {
        nodes { name }
      }
      closingIssuesReferences(first: 20) @include(if: $includeDetail) {
        nodes { number }
      }
      latestReviews(first: 100) @include(if: $includeDetail) {
        nodes {
          state
          author { login }
        }
        pageInfo {
          hasNextPage
        }
      }
      comments(last: 50) @include(if: $includeDetail) {
        nodes {
          author { login }
          body
          updatedAt
        }
      }
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

// maxReviewThreadPages caps reviewThreads pagination (same contract as former reviews package).
const maxReviewThreadPages = 500

//nolint:govet // fieldalignment: keep aligned with GraphQL response shape
type prContextResponse struct {
	Repository *struct {
		PullRequest *prPullRequestNode `json:"pullRequest"`
	} `json:"repository"`
}

type prPullRequestNode struct {
	State *string `json:"state"`
	// IsDraft is returned as JSON boolean when @include detail.
	IsDraft                 *bool              `json:"isDraft"`
	Title                   *string            `json:"title"`
	Mergeable               *string            `json:"mergeable"`
	MergeStateStatus        *string            `json:"mergeStateStatus"`
	BaseRefName             *string            `json:"baseRefName"`
	HeadRefOid              *string            `json:"headRefOid"`
	Labels                  *labelsConn        `json:"labels"`
	ClosingIssuesReferences *closingIssuesConn `json:"closingIssuesReferences"`
	LatestReviews           *latestReviewsConn `json:"latestReviews"`
	Comments                *commentsConn      `json:"comments"`
	ReviewThreads           *reviewThreadsConn `json:"reviewThreads"`
}

type labelsConn struct {
	Nodes []struct {
		Name string `json:"name"`
	} `json:"nodes"`
}

type closingIssuesConn struct {
	Nodes []struct {
		Number int `json:"number"`
	} `json:"nodes"`
}

//nolint:govet // fieldalignment: keep aligned with GraphQL response shape
type latestReviewsConn struct {
	Nodes []struct {
		State  string `json:"state"`
		Author *struct {
			Login string `json:"login"`
		} `json:"author"`
	} `json:"nodes"`
	PageInfo struct {
		HasNextPage bool `json:"hasNextPage"`
	} `json:"pageInfo"`
}

type commentsConn struct {
	Nodes []struct {
		Author *struct {
			Login string `json:"login"`
		} `json:"author"`
		Body      string `json:"body"`
		UpdatedAt string `json:"updatedAt"`
	} `json:"nodes"`
}

//nolint:govet // fieldalignment: keep aligned with GraphQL response shape
type reviewThreadsConn struct {
	Nodes []struct {
		IsResolved bool `json:"isResolved"`
	} `json:"nodes"`
	PageInfo struct {
		HasNextPage bool   `json:"hasNextPage"`
		EndCursor   string `json:"endCursor"`
	} `json:"pageInfo"`
}

// Collect returns pull-request detail fields to merge into signals.github.pull_requests and
// the inner map for signals.github.reviews. pullDetail is nil when there is nothing to merge.
// reviewsInner is always non-nil when err is nil (empty maps on PR≤0 or missing PR).
func Collect(ctx context.Context, c *githubapi.Client, owner, repo string, prNumber int, tracked []TrackedReviewer) (pullDetail map[string]any, reviewsInner map[string]any, err error) {
	if c == nil {
		return nil, nil, fmt.Errorf("nil client")
	}
	reviewsInner = emptyReviewsInner()
	if prNumber <= 0 {
		return nil, reviewsInner, nil
	}
	firstPR, total, unresolved, incomplete, err := paginatePRContext(ctx, c, owner, repo, prNumber)
	if err != nil {
		return nil, nil, err
	}
	if firstPR == nil {
		return nil, reviewsInner, nil
	}
	pullDetail = buildPullDetail(firstPR)
	decisions := buildReviewDecisions(firstPR.LatestReviews)
	for k, v := range decisions {
		reviewsInner[k] = v
	}
	reviewsInner["review_threads_total"] = total
	reviewsInner["review_threads_unresolved"] = unresolved
	reviewsInner["pagination_incomplete"] = incomplete
	reviewsInner["tracked_reviewer_status"] = buildTrackedReviewerStatus(firstPR, tracked)
	return pullDetail, reviewsInner, nil
}

func paginatePRContext(ctx context.Context, c *githubapi.Client, owner, repo string, prNumber int) (firstPR *prPullRequestNode, total, unresolved int, incomplete bool, err error) {
	var cursor any
	for page := 0; page < maxReviewThreadPages; page++ {
		includeDetail := page == 0
		vars := map[string]any{
			"owner":         owner,
			"name":          repo,
			"number":        prNumber,
			"cursor":        cursor,
			"includeDetail": includeDetail,
		}
		var data prContextResponse
		if err := c.PostGraphQL(ctx, prContextQuery, vars, &data); err != nil {
			return nil, 0, 0, false, err
		}
		if data.Repository == nil || data.Repository.PullRequest == nil {
			return nil, 0, 0, false, nil
		}
		pr := data.Repository.PullRequest
		if includeDetail {
			firstPR = pr
		}
		rt := pr.ReviewThreads
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
		end := strings.TrimSpace(rt.PageInfo.EndCursor)
		if end == "" {
			incomplete = true
			break
		}
		cursor = end
	}
	return firstPR, total, unresolved, incomplete, nil
}

func emptyReviewsInner() map[string]any {
	return map[string]any{
		"review_threads_total":               0,
		"review_threads_unresolved":          0,
		"pagination_incomplete":              false,
		"review_decisions_total":             0,
		"review_decisions_approved":          0,
		"review_decisions_changes_requested": 0,
		"review_decisions_truncated":         false,
		"tracked_reviewer_status":            []any{},
	}
}

func buildPullDetail(pr *prPullRequestNode) map[string]any {
	m := make(map[string]any)
	if pr.State != nil {
		m["state"] = strings.ToLower(strings.TrimSpace(*pr.State))
	}
	if pr.IsDraft != nil {
		m["draft"] = *pr.IsDraft
	}
	if pr.Title != nil {
		m["title"] = *pr.Title
	}
	if pr.BaseRefName != nil {
		m["base_ref"] = *pr.BaseRefName
	}
	if pr.HeadRefOid != nil {
		m["head_sha"] = *pr.HeadRefOid
	}
	if pr.Mergeable != nil {
		m["mergeable"] = mergeableToSignal(*pr.Mergeable)
	}
	if pr.MergeStateStatus != nil {
		m["merge_state_status"] = strings.ToLower(strings.TrimSpace(*pr.MergeStateStatus))
	}
	if pr.Labels != nil {
		var names []any
		for _, n := range pr.Labels.Nodes {
			if n.Name != "" {
				names = append(names, n.Name)
			}
		}
		m["labels"] = names
	}
	if pr.ClosingIssuesReferences != nil {
		var nums []any
		for _, n := range pr.ClosingIssuesReferences.Nodes {
			nums = append(nums, n.Number)
		}
		m["closing_issue_numbers"] = nums
	}
	return m
}

func mergeableToSignal(gql string) string {
	switch strings.ToUpper(strings.TrimSpace(gql)) {
	case "MERGEABLE":
		return "mergeable"
	case "CONFLICTING":
		return "conflicting"
	default:
		return "unknown"
	}
}

func buildReviewDecisions(lr *latestReviewsConn) map[string]any {
	out := map[string]any{
		"review_decisions_total":             0,
		"review_decisions_approved":          0,
		"review_decisions_changes_requested": 0,
		"review_decisions_truncated":         false,
	}
	if lr == nil {
		return out
	}
	approved := 0
	changes := 0
	for _, n := range lr.Nodes {
		switch strings.ToUpper(strings.TrimSpace(n.State)) {
		case "APPROVED":
			approved++
		case "CHANGES_REQUESTED":
			changes++
		}
	}
	total := len(lr.Nodes)
	out["review_decisions_total"] = total
	out["review_decisions_approved"] = approved
	out["review_decisions_changes_requested"] = changes
	out["review_decisions_truncated"] = lr.PageInfo.HasNextPage
	return out
}

func buildTrackedReviewerStatus(pr *prPullRequestNode, tracked []TrackedReviewer) []any {
	if len(tracked) == 0 {
		return []any{}
	}
	reviewByLogin := map[string]string{}
	if pr.LatestReviews != nil {
		for _, n := range pr.LatestReviews.Nodes {
			if n.Author == nil {
				continue
			}
			login := n.Author.Login
			if login != "" {
				reviewByLogin[strings.ToLower(login)] = n.State
			}
		}
	}
	var nodes []struct {
		Author *struct {
			Login string `json:"login"`
		} `json:"author"`
		Body      string `json:"body"`
		UpdatedAt string `json:"updatedAt"`
	}
	if pr.Comments != nil {
		nodes = pr.Comments.Nodes
	}
	out := make([]any, 0, len(tracked))
	for _, tr := range tracked {
		login := strings.TrimSpace(tr.Login)
		if login == "" {
			continue
		}
		key := strings.ToLower(login)
		state, hasRev := reviewByLogin[key]
		if !hasRev {
			state = ""
		}
		body, updated := latestCommentForLogin(nodes, login)
		lower := strings.ToLower(body)
		status := map[string]any{
			"login":                        login,
			"has_review":                   hasRev,
			"review_state":                 state,
			"latest_comment_at":            updated,
			"contains_rate_limit":          strings.Contains(lower, "rate limit"),
			"contains_review_paused":       strings.Contains(lower, "review paused") || strings.Contains(lower, "review pause"),
			"contains_review_failed":       strings.Contains(lower, "review failed"),
			"rate_limit_remaining_seconds": 0,
		}
		if extra := applyEnrichments(body, tr.Enrich); len(extra) > 0 {
			for k, v := range extra {
				status[k] = v
			}
		}
		out = append(out, status)
	}
	return out
}

func latestCommentForLogin(nodes []struct {
	Author *struct {
		Login string `json:"login"`
	} `json:"author"`
	Body      string `json:"body"`
	UpdatedAt string `json:"updatedAt"`
}, wantLogin string) (body, updatedAt string) {
	var best string
	for _, n := range nodes {
		if n.Author == nil {
			continue
		}
		if !strings.EqualFold(n.Author.Login, wantLogin) {
			continue
		}
		// Lexicographic compare on RFC3339 works for picking latest.
		if n.UpdatedAt >= best {
			best = n.UpdatedAt
			body = n.Body
			updatedAt = n.UpdatedAt
		}
	}
	return body, updatedAt
}
