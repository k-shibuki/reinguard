// Package prquery runs a unified GitHub GraphQL query for the current PR: detail, review
// threads, latest review decisions, linked issues, and PR comments for configured bot reviewers
// (ADR-0006, ADR-0012).
package prquery

import (
	"context"
	"fmt"
	"strings"

	"github.com/k-shibuki/reinguard/internal/githubapi"
)

// BotReviewer is one github provider option entry for bot / AI reviewer observation.
type BotReviewer struct {
	ID       string
	Login    string
	Enrich   []string
	Required bool
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
          body
          author { login }
          commit { oid }
        }
        pageInfo {
          hasNextPage
        }
      }
      comments(last: 100) @include(if: $includeDetail) {
        pageInfo {
          hasPreviousPage
          startCursor
        }
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

// maxCommentBackPages caps older PR comment fetches when walking back for configured bot logins.
const maxCommentBackPages = 30

const prCommentsPageQuery = `query PRCommentPage($owner: String!, $name: String!, $number: Int!, $before: String!) {
  repository(owner: $owner, name: $name) {
    pullRequest(number: $number) {
      comments(last: 100, before: $before) {
        pageInfo {
          hasPreviousPage
          startCursor
        }
        nodes {
          author { login }
          body
          updatedAt
        }
      }
    }
  }
}`

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
		Body   string `json:"body"`
		Author *struct {
			Login string `json:"login"`
		} `json:"author"`
		Commit *struct {
			Oid string `json:"oid"`
		} `json:"commit"`
	} `json:"nodes"`
	PageInfo struct {
		HasNextPage bool `json:"hasNextPage"`
	} `json:"pageInfo"`
}

// prCommentNode is one issue comment on the pull request timeline (GraphQL shape).
//
//nolint:govet // fieldalignment: GraphQL response shape
type prCommentNode struct {
	Author *struct {
		Login string `json:"login"`
	} `json:"author"`
	Body      string `json:"body"`
	UpdatedAt string `json:"updatedAt"`
}

//nolint:govet // fieldalignment: keep pageInfo before nodes for padding; matches GraphQL shape
type commentsConn struct {
	PageInfo struct {
		HasPreviousPage bool   `json:"hasPreviousPage"`
		StartCursor     string `json:"startCursor"`
	} `json:"pageInfo"`
	Nodes []prCommentNode `json:"nodes"`
}

//nolint:govet // fieldalignment: GraphQL response shape
type prCommentsPageResponse struct {
	Repository *struct {
		PullRequest *struct {
			Comments *commentsConn `json:"comments"`
		} `json:"pullRequest"`
	} `json:"repository"`
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
func Collect(ctx context.Context, c *githubapi.Client, owner, repo string, prNumber int, bots []BotReviewer) (pullDetail map[string]any, reviewsInner map[string]any, err error) {
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
	commentNodes, err := mergeCommentNodesForBots(ctx, c, owner, repo, prNumber, firstPR.Comments, bots)
	if err != nil {
		return nil, nil, err
	}
	statusList := buildBotReviewerStatus(firstPR, bots, commentNodes)
	reviewsInner["bot_reviewer_status"] = statusList
	var headSHA string
	if firstPR.HeadRefOid != nil {
		headSHA = *firstPR.HeadRefOid
	}
	reviewsInner["bot_review_diagnostics"] = ComputeBotReviewDiagnostics(statusList, headSHA)
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
		"bot_reviewer_status":                []any{},
		"bot_review_diagnostics": map[string]any{
			"bot_review_completed":        true,
			"bot_review_pending":          false,
			"bot_review_terminal":         true,
			"bot_review_failed":           false,
			"duplicate_findings_detected": false,
		},
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
		names := make([]any, 0, len(pr.Labels.Nodes))
		for _, n := range pr.Labels.Nodes {
			if n.Name != "" {
				names = append(names, n.Name)
			}
		}
		m["labels"] = names
	}
	if pr.ClosingIssuesReferences != nil {
		nums := make([]any, 0, len(pr.ClosingIssuesReferences.Nodes))
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

// normalizeGitHubActorLogin maps login variants to one comparison key. GitHub App bots
// may appear as "name[bot]" in config/REST and as "name" on GraphQL issue comments / reviews.
func normalizeGitHubActorLogin(login string) string {
	s := strings.TrimSpace(strings.ToLower(login))
	s = strings.TrimSuffix(s, "[bot]")
	return strings.TrimSpace(s)
}

func mergeCommentNodesForBots(ctx context.Context, c *githubapi.Client, owner, repo string, prNumber int, seed *commentsConn, bots []BotReviewer) ([]prCommentNode, error) {
	if len(bots) == 0 {
		if seed == nil {
			return nil, nil
		}
		out := make([]prCommentNode, len(seed.Nodes))
		copy(out, seed.Nodes)
		return out, nil
	}
	want := make(map[string]struct{})
	for _, br := range bots {
		if lg := strings.TrimSpace(br.Login); lg != "" {
			want[normalizeGitHubActorLogin(lg)] = struct{}{}
		}
	}
	var merged []prCommentNode
	if seed != nil {
		merged = append(merged, seed.Nodes...)
	}
	seen := botKeysSeenInComments(merged, want)
	hasPrev := seed != nil && seed.PageInfo.HasPreviousPage
	before := ""
	if seed != nil {
		before = strings.TrimSpace(seed.PageInfo.StartCursor)
	}
	for page := 0; page < maxCommentBackPages && hasPrev && !allBotsSeen(seen, want); page++ {
		if before == "" {
			break
		}
		conn, err := fetchPRCommentsPage(ctx, c, owner, repo, prNumber, before)
		if err != nil {
			return nil, err
		}
		if conn == nil {
			break
		}
		merged = append(merged, conn.Nodes...)
		markBotsSeen(conn.Nodes, want, seen)
		hasPrev = conn.PageInfo.HasPreviousPage
		before = strings.TrimSpace(conn.PageInfo.StartCursor)
	}
	return merged, nil
}

func fetchPRCommentsPage(ctx context.Context, c *githubapi.Client, owner, repo string, prNumber int, before string) (*commentsConn, error) {
	vars := map[string]any{"owner": owner, "name": repo, "number": prNumber, "before": before}
	var data prCommentsPageResponse
	if err := c.PostGraphQL(ctx, prCommentsPageQuery, vars, &data); err != nil {
		return nil, err
	}
	if data.Repository == nil || data.Repository.PullRequest == nil {
		return nil, nil
	}
	return data.Repository.PullRequest.Comments, nil
}

func botKeysSeenInComments(nodes []prCommentNode, want map[string]struct{}) map[string]bool {
	seen := make(map[string]bool)
	markBotsSeen(nodes, want, seen)
	return seen
}

func markBotsSeen(nodes []prCommentNode, want map[string]struct{}, seen map[string]bool) {
	for _, n := range nodes {
		if n.Author == nil {
			continue
		}
		k := normalizeGitHubActorLogin(n.Author.Login)
		if _, ok := want[k]; ok {
			seen[k] = true
		}
	}
}

func allBotsSeen(seen map[string]bool, want map[string]struct{}) bool {
	for k := range want {
		if !seen[k] {
			return false
		}
	}
	return true
}

// reviewFailedFromComment detects terminal bot failure cues in PR timeline comments,
// including head-moved / voided review messages documented in review--bot-operations.md.
func reviewFailedFromComment(lower string) bool {
	if strings.Contains(lower, "review failed") {
		return true
	}
	if strings.Contains(lower, "head commit changed during the review") {
		return true
	}
	if strings.Contains(lower, "review was voided") || strings.Contains(lower, "voided review") {
		return true
	}
	return false
}

func buildBotReviewerStatus(pr *prPullRequestNode, bots []BotReviewer, commentNodes []prCommentNode) []any {
	if len(bots) == 0 {
		return []any{}
	}
	type reviewInfo struct {
		State     string
		CommitOid string
		Body      string
	}
	reviewByLogin := map[string]reviewInfo{}
	if pr.LatestReviews != nil {
		for _, n := range pr.LatestReviews.Nodes {
			if n.Author == nil {
				continue
			}
			login := n.Author.Login
			if login == "" {
				continue
			}
			ri := reviewInfo{State: n.State, Body: n.Body}
			if n.Commit != nil {
				ri.CommitOid = n.Commit.Oid
			}
			reviewByLogin[normalizeGitHubActorLogin(login)] = ri
		}
	}
	nodes := commentNodes
	out := make([]any, 0, len(bots))
	for _, br := range bots {
		login := strings.TrimSpace(br.Login)
		if login == "" {
			continue
		}
		key := normalizeGitHubActorLogin(login)
		ri, hasRev := reviewByLogin[key]
		state := ri.State
		if !hasRev {
			state = ""
		}
		body, updated := latestCommentForLogin(nodes, login)
		lower := strings.ToLower(body)
		status := map[string]any{
			"id":                     strings.TrimSpace(br.ID),
			"login":                  login,
			"required":               br.Required,
			"has_review":             hasRev,
			"review_state":           state,
			"review_commit_sha":      ri.CommitOid,
			"latest_comment_at":      updated,
			"contains_rate_limit":    strings.Contains(lower, "rate limit"),
			"contains_review_paused": strings.Contains(lower, "review paused") || strings.Contains(lower, "review pause"),
			"contains_review_failed": reviewFailedFromComment(lower),
		}
		if extra := applyEnrichments(body, br.Enrich); len(extra) > 0 {
			for k, v := range extra {
				status[k] = v
			}
		}
		if extra := applyReviewBodyEnrichments(ri.Body, br.Enrich); len(extra) > 0 {
			for k, v := range extra {
				status[k] = v
			}
		}
		status["status"] = ClassifyBotStatus(status, br.Enrich)
		out = append(out, status)
	}
	return out
}

func latestCommentForLogin(nodes []prCommentNode, wantLogin string) (body, updatedAt string) {
	var best string
	for _, n := range nodes {
		if n.Author == nil {
			continue
		}
		if normalizeGitHubActorLogin(n.Author.Login) != normalizeGitHubActorLogin(wantLogin) {
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
