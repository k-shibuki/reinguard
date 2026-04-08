// Package prquery runs a unified GitHub GraphQL query for the current PR: detail, review
// threads, latest review decisions, linked issues, and PR comments for configured bot reviewers
// (ADR-0006, ADR-0012).
package prquery

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/k-shibuki/reinguard/internal/githubapi"
)

// BotReviewer is one github provider option entry for bot / AI reviewer observation.
type BotReviewer struct {
	ID       string
	Login    string
	Enrich   []string
	Required bool
}

// enrichmentNameCoderabbit is the registered enrichment plugin id for CodeRabbit (reinguard.yaml enrich[]).
const enrichmentNameCoderabbit = "coderabbit"

func enrichIncludesCoderabbit(names []string) bool {
	for _, n := range names {
		if strings.TrimSpace(n) == enrichmentNameCoderabbit {
			return true
		}
	}
	return false
}

// applyCoderabbitStatusClassBasis sets status_class_basis when CodeRabbit enrichment is enabled.
func applyCoderabbitStatusClassBasis(status map[string]any, enrich []string) {
	if !enrichIncludesCoderabbit(enrich) {
		return
	}
	_, basis := classifyCoderabbitStatusWithBasis(status)
	status["status_class_basis"] = basis
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
      headRefName @include(if: $includeDetail)
      headRefOid @include(if: $includeDetail)
      headRepository @include(if: $includeDetail) {
        name
        owner { login }
      }
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
          id
          isResolved
          isOutdated
          comments(first: 1) {
            nodes {
              databaseId
              body
              path
              line
              originalLine
              startLine
              originalStartLine
              author { login }
              commit { oid }
              originalCommit { oid }
            }
          }
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

//nolint:govet // fieldalignment: GraphQL response shape
type headRepositoryNode struct {
	Name  string `json:"name"`
	Owner *struct {
		Login string `json:"login"`
	} `json:"owner"`
}

type prPullRequestNode struct {
	State *string `json:"state"`
	// IsDraft is returned as JSON boolean when @include detail.
	IsDraft                 *bool               `json:"isDraft"`
	Title                   *string             `json:"title"`
	Mergeable               *string             `json:"mergeable"`
	MergeStateStatus        *string             `json:"mergeStateStatus"`
	BaseRefName             *string             `json:"baseRefName"`
	HeadRefName             *string             `json:"headRefName"`
	HeadRefOid              *string             `json:"headRefOid"`
	HeadRepository          *headRepositoryNode `json:"headRepository"`
	Labels                  *labelsConn         `json:"labels"`
	ClosingIssuesReferences *closingIssuesConn  `json:"closingIssuesReferences"`
	LatestReviews           *latestReviewsConn  `json:"latestReviews"`
	Comments                *commentsConn       `json:"comments"`
	ReviewThreads           *reviewThreadsConn  `json:"reviewThreads"`
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
	Nodes    []reviewThreadNode `json:"nodes"`
	PageInfo struct {
		HasNextPage bool   `json:"hasNextPage"`
		EndCursor   string `json:"endCursor"`
	} `json:"pageInfo"`
}

//nolint:govet // fieldalignment: GraphQL response shape
type reviewThreadNode struct {
	ID         string                    `json:"id"`
	IsResolved bool                      `json:"isResolved"`
	IsOutdated bool                      `json:"isOutdated"`
	Comments   *reviewThreadCommentsConn `json:"comments"`
}

type reviewThreadCommentsConn struct {
	Nodes []reviewThreadCommentNode `json:"nodes"`
}

//nolint:govet // fieldalignment: GraphQL response shape
type reviewThreadCommentNode struct {
	DatabaseID        int    `json:"databaseId"`
	Body              string `json:"body"`
	Path              string `json:"path"`
	Line              *int   `json:"line"`
	OriginalLine      *int   `json:"originalLine"`
	StartLine         *int   `json:"startLine"`
	OriginalStartLine *int   `json:"originalStartLine"`
	Author            *struct {
		Login string `json:"login"`
	} `json:"author"`
	Commit *struct {
		Oid string `json:"oid"`
	} `json:"commit"`
	OriginalCommit *struct {
		Oid string `json:"oid"`
	} `json:"originalCommit"`
}

// CollectOptions configures optional behavior for CollectWithOptions.
type CollectOptions struct {
	// Now is the observation wall time used to age-adjust rate_limit_remaining_seconds
	// against status_comment_at. If nil, time.Now() is used.
	Now *time.Time
}

// Collect returns pull-request detail fields to merge into signals.github.pull_requests and
// the inner map for signals.github.reviews. pullDetail is nil when there is nothing to merge.
// reviewsInner is always non-nil when err is nil (empty maps on PR≤0 or missing PR).
func Collect(ctx context.Context, c *githubapi.Client, owner, repo string, prNumber int, bots []BotReviewer) (pullDetail map[string]any, reviewsInner map[string]any, err error) {
	return CollectWithOptions(ctx, c, owner, repo, prNumber, bots, nil)
}

// CollectWithOptions is like Collect but accepts optional opts (e.g. fixed clock for tests).
func CollectWithOptions(ctx context.Context, c *githubapi.Client, owner, repo string, prNumber int, bots []BotReviewer, opts *CollectOptions) (pullDetail map[string]any, reviewsInner map[string]any, err error) {
	if c == nil {
		return nil, nil, fmt.Errorf("nil client")
	}
	reviewsInner = emptyReviewsInner()
	if prNumber <= 0 {
		return nil, reviewsInner, nil
	}
	firstPR, inbox, total, unresolved, incomplete, err := paginatePRContext(ctx, c, owner, repo, prNumber)
	if err != nil {
		return nil, nil, err
	}
	if firstPR == nil {
		return nil, reviewsInner, nil
	}
	now := time.Now()
	if opts != nil && opts.Now != nil {
		now = opts.Now.UTC()
	}
	pullDetail = buildPullDetail(firstPR)
	decisions := buildReviewDecisions(firstPR.LatestReviews)
	for k, v := range decisions {
		reviewsInner[k] = v
	}
	reviewsInner["review_threads_total"] = total
	reviewsInner["review_threads_unresolved"] = unresolved
	reviewsInner["pagination_incomplete"] = incomplete
	if inbox == nil {
		inbox = []any{}
	}
	reviewsInner["review_inbox"] = inbox
	commentNodes, conversationIncomplete, err := mergeCommentNodesForBots(ctx, c, owner, repo, prNumber, firstPR.Comments, bots)
	if err != nil {
		return nil, nil, err
	}
	reviewsInner["conversation_comments"] = buildConversationCommentsReadModel(commentNodes)
	reviewsInner["conversation_comments_incomplete"] = conversationIncomplete
	statusList := buildBotReviewerStatus(firstPR, bots, commentNodes, now)
	reviewsInner["bot_reviewer_status"] = statusList
	var headSHA string
	if firstPR.HeadRefOid != nil {
		headSHA = *firstPR.HeadRefOid
	}
	reviewsInner["bot_review_diagnostics"] = ComputeBotReviewDiagnostics(statusList, headSHA, conversationIncomplete)
	return pullDetail, reviewsInner, nil
}

// paginatePRContext walks review-thread GraphQL pages until the inbox is complete or the
// page budget is hit; incomplete is true when older thread pages were not fetched.
func paginatePRContext(ctx context.Context, c *githubapi.Client, owner, repo string, prNumber int) (firstPR *prPullRequestNode, inbox []any, total, unresolved int, incomplete bool, err error) {
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
			return nil, nil, 0, 0, false, err
		}
		if data.Repository == nil || data.Repository.PullRequest == nil {
			return nil, nil, 0, 0, false, nil
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
			if entry := buildReviewInboxEntry(n); entry != nil {
				inbox = append(inbox, entry)
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
	return firstPR, inbox, total, unresolved, incomplete, nil
}

// emptyReviewsInner returns a zeroed github.reviews subtree used when no PR exists.
func emptyReviewsInner() map[string]any {
	return map[string]any{
		"review_threads_total":               0,
		"review_threads_unresolved":          0,
		"pagination_incomplete":              false,
		"review_inbox":                       []any{},
		"conversation_comments":              []any{},
		"conversation_comments_incomplete":   false,
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
			"bot_review_stale":            false,
			"duplicate_findings_detected": false,
			"non_thread_findings_present": false,
		},
	}
}

// applyHeadRepositorySignals copies fork head owner/name from GraphQL into pull-request signals.
func applyHeadRepositorySignals(pr *prPullRequestNode, m map[string]any) {
	if pr == nil || pr.HeadRepository == nil {
		return
	}
	if pr.HeadRepository.Owner != nil && strings.TrimSpace(pr.HeadRepository.Owner.Login) != "" {
		m["head_repo_owner"] = strings.TrimSpace(pr.HeadRepository.Owner.Login)
	}
	if strings.TrimSpace(pr.HeadRepository.Name) != "" {
		m["head_repo_name"] = strings.TrimSpace(pr.HeadRepository.Name)
	}
}

// buildPullDetail maps GraphQL PR fields into the flat pull-request signal map (state, refs,
// mergeability, labels, and linked closing issues).
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
	if pr.HeadRefName != nil {
		m["head_ref"] = *pr.HeadRefName
	}
	if pr.HeadRefOid != nil {
		m["head_sha"] = *pr.HeadRefOid
	}
	applyHeadRepositorySignals(pr, m)
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

// mergeableToSignal normalizes GitHub PullRequest.mergeable enum strings for observation signals.
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

// buildConversationCommentsReadModel turns issue-comment nodes into the JSON read model for github.reviews.conversation_comments.
func buildConversationCommentsReadModel(nodes []prCommentNode) []any {
	out := make([]any, 0, len(nodes))
	for _, n := range nodes {
		entry := map[string]any{"body": n.Body}
		if n.Author != nil && strings.TrimSpace(n.Author.Login) != "" {
			entry["author"] = strings.TrimSpace(n.Author.Login)
		}
		if strings.TrimSpace(n.UpdatedAt) != "" {
			entry["updated_at"] = n.UpdatedAt
		}
		out = append(out, entry)
	}
	return out
}

// applyFindingConversationCountIfNeeded sets finding-conversation counts from the first
// configured enrichment that implements findingConversationCounter; later enrich[] names are ignored.
func applyFindingConversationCountIfNeeded(status map[string]any, enrich []string, nodes []prCommentNode, login string) {
	for _, name := range enrich {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		e, ok := enrichmentByNameLocked(name)
		if !ok {
			continue
		}
		counter, ok := e.(findingConversationCounter)
		if !ok {
			continue
		}
		count := counter.CountFindingConversationComments(nodes, login)
		status["finding_conversation_comments_count"] = count
		if name == enrichmentNameCoderabbit {
			status["cr_finding_conversation_comments_count"] = count
		}
		return
	}
}

func countCoderabbitFindingConversationComments(nodes []prCommentNode, wantLogin string) int {
	// Conservative aggregate: counts issue comments that look like findings for merge gating.
	// User disposition replies on the PR do not decrement this counter; reviewers use the
	// disposition workflow for non-thread closure (see review--consensus-protocol.md).
	key := normalizeGitHubActorLogin(wantLogin)
	var n int
	for _, cn := range nodes {
		if cn.Author == nil {
			continue
		}
		if normalizeGitHubActorLogin(cn.Author.Login) != key {
			continue
		}
		if IsCoderabbitFindingConversationComment(cn.Body) {
			n++
		}
	}
	return n
}

// buildReviewDecisions aggregates latest review submission states (approve vs changes_requested) and truncation.
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

// buildReviewInboxEntry builds one unresolved-thread entry for the review inbox, or nil if resolved or missing id.
func buildReviewInboxEntry(thread reviewThreadNode) map[string]any {
	if thread.IsResolved {
		return nil
	}
	threadID := strings.TrimSpace(thread.ID)
	if threadID == "" {
		return nil
	}
	entry := map[string]any{
		"thread_id":   threadID,
		"is_outdated": thread.IsOutdated,
	}
	if thread.Comments == nil || len(thread.Comments.Nodes) == 0 {
		return entry
	}
	root := thread.Comments.Nodes[0]
	putReviewInboxInt(entry, "root_comment_id", root.DatabaseID)
	putReviewInboxString(entry, "body", root.Body)
	putReviewInboxString(entry, "path", root.Path)
	linePtr := root.Line
	if linePtr == nil {
		linePtr = root.OriginalLine
	}
	putReviewInboxOptionalInt(entry, "line", linePtr)
	putReviewInboxOptionalInt(entry, "original_line", root.OriginalLine)
	startPtr := root.StartLine
	if startPtr == nil {
		startPtr = root.OriginalStartLine
	}
	putReviewInboxOptionalInt(entry, "start_line", startPtr)
	putReviewInboxOptionalInt(entry, "original_start_line", root.OriginalStartLine)
	if root.Author != nil {
		putReviewInboxString(entry, "author", root.Author.Login)
	}
	var commitOid string
	if root.Commit != nil && strings.TrimSpace(root.Commit.Oid) != "" {
		commitOid = root.Commit.Oid
	} else if root.OriginalCommit != nil {
		commitOid = root.OriginalCommit.Oid
	}
	putReviewInboxString(entry, "commit_sha", commitOid)
	if root.OriginalCommit != nil {
		putReviewInboxString(entry, "original_commit_sha", root.OriginalCommit.Oid)
	}
	return entry
}

// putReviewInboxInt adds key only for positive values, which matches GitHub database IDs.
func putReviewInboxInt(entry map[string]any, key string, value int) {
	if value > 0 {
		entry[key] = value
	}
}

func putReviewInboxOptionalInt(entry map[string]any, key string, value *int) {
	if value != nil {
		entry[key] = *value
	}
}

func putReviewInboxString(entry map[string]any, key, value string) {
	if trimmed := strings.TrimSpace(value); trimmed != "" {
		entry[key] = trimmed
	}
}

// normalizeGitHubActorLogin maps login variants to one comparison key. GitHub App bots
// may appear as "name[bot]" in config/REST and as "name" on GraphQL issue comments / reviews.
func normalizeGitHubActorLogin(login string) string {
	s := strings.TrimSpace(strings.ToLower(login))
	s = strings.TrimSuffix(s, "[bot]")
	return strings.TrimSpace(s)
}

// mergeCommentNodesForBots prepends older issue-comment pages while back-pagination budget
// remains; the bool is true when older pages may remain beyond what we merged.
func mergeCommentNodesForBots(ctx context.Context, c *githubapi.Client, owner, repo string, prNumber int, seed *commentsConn, bots []BotReviewer) ([]prCommentNode, bool, error) {
	if len(bots) == 0 {
		if seed == nil {
			return nil, false, nil
		}
		out := make([]prCommentNode, len(seed.Nodes))
		copy(out, seed.Nodes)
		incomplete := seed.PageInfo.HasPreviousPage
		return out, incomplete, nil
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
	for page := 0; page < maxCommentBackPages && hasPrev; page++ {
		if before == "" {
			break
		}
		conn, err := fetchPRCommentsPage(ctx, c, owner, repo, prNumber, before)
		if err != nil {
			return nil, false, err
		}
		if conn == nil {
			break
		}
		if len(conn.Nodes) > 0 {
			older := make([]prCommentNode, 0, len(conn.Nodes)+len(merged))
			older = append(older, conn.Nodes...)
			merged = append(older, merged...)
		}
		markBotsSeen(conn.Nodes, want, seen)
		hasPrev = conn.PageInfo.HasPreviousPage
		before = strings.TrimSpace(conn.PageInfo.StartCursor)
	}
	// hasPrev means older comment pages exist beyond what we merged; counts may be partial.
	return merged, hasPrev, nil
}

// fetchPRCommentsPage loads one page of PR issue comments before the given cursor.
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

// botKeysSeenInComments reports which normalized bot logins from want appear in nodes.
func botKeysSeenInComments(nodes []prCommentNode, want map[string]struct{}) map[string]bool {
	seen := make(map[string]bool)
	markBotsSeen(nodes, want, seen)
	return seen
}

// markBotsSeen records normalized author logins from nodes that match want into seen.
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

// applyReviewCommitSHAFromEnrichedComment sets review_commit_sha from an enriched reviewed head
// when GraphQL did not return a review (comment-only bot completion), or when latestReviews still
// points at an older review commit while the selected status comment names the current PR head in
// its reviewed range (CodeRabbit summarize / clean-bill edits).
func applyReviewCommitSHAFromEnrichedComment(status map[string]any, headSHA string) {
	enriched := strings.TrimSpace(signalString(status, "reviewed_head_sha"))
	if enriched == "" {
		enriched = strings.TrimSpace(signalString(status, "cr_reviewed_head_sha"))
	}
	cur := strings.TrimSpace(signalString(status, "review_commit_sha"))

	if enriched != "" && headSHA != "" && strings.EqualFold(enriched, headSHA) {
		status["review_commit_sha"] = enriched
		return
	}
	if cur == "" && enriched != "" {
		status["review_commit_sha"] = enriched
	}
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

// genericIssueCommentMaxTier assigns coarse tiers for non-CodeRabbit enrichment bots.
func genericIssueCommentMaxTier(body string) int {
	lower := strings.ToLower(body)
	t := 0
	if strings.Contains(lower, "rate limit") {
		t = max(t, 4)
	}
	if strings.Contains(lower, "review paused") || strings.Contains(lower, "review pause") {
		t = max(t, 3)
	}
	if reviewFailedFromComment(lower) {
		t = max(t, 2)
	}
	return t
}

// commentMaxTier returns the semantic tier for body using the first enrichment that implements
// issueCommentTierer, otherwise genericIssueCommentMaxTier.
func commentMaxTier(body string, enrichNames []string) int {
	for _, name := range enrichNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		e, ok := enrichmentByNameLocked(name)
		if !ok {
			continue
		}
		tierer, ok := e.(issueCommentTierer)
		if !ok {
			continue
		}
		return tierer.CommentMaxTier(body)
	}
	return genericIssueCommentMaxTier(body)
}

// coerceInt parses common JSON numeric shapes into int for status maps.
func coerceInt(v any) (int, bool) {
	switch x := v.(type) {
	case int:
		return x, true
	case int64:
		return int(x), true
	case float64:
		return int(x), true
	default:
		return 0, false
	}
}

// adjustRateLimitRemainingForStatusCommentAge replaces rate_limit_remaining_seconds with
// max(0, parsed_from_body - elapsed_since_status_comment_at). The parsed duration in the
// comment body is relative to the selected status comment's updatedAt (status_comment_at).
// When CodeRabbit edits an issue comment in place, GitHub advances updatedAt; the next
// observation uses the new body and new status_comment_at, so the cooldown re-anchors.
func adjustRateLimitRemainingForStatusCommentAge(status map[string]any, now time.Time) {
	raw, ok := status["rate_limit_remaining_seconds"]
	if !ok || raw == nil {
		return
	}
	parsed, ok := coerceInt(raw)
	if !ok || parsed < 0 {
		return
	}
	sat, _ := status["status_comment_at"].(string)
	sat = strings.TrimSpace(sat)
	if sat == "" {
		return
	}
	t, err := time.Parse(time.RFC3339, sat)
	if err != nil {
		return
	}
	elapsed := int(now.Sub(t.UTC()).Seconds())
	if elapsed < 0 {
		elapsed = 0
	}
	rem := parsed - elapsed
	if rem < 0 {
		rem = 0
	}
	status["rate_limit_remaining_seconds"] = rem
}

// selectStatusCommentForLogin picks the PR issue comment whose body drives bot status
// (contains_* and issue-comment enrichment). It prefers the highest semantic tier, then the
// newest updatedAt within that tier, so an edited Review Status comment is not shadowed by a
// later short acknowledgment comment without status markers.
func selectStatusCommentForLogin(nodes []prCommentNode, wantLogin string, enrichNames []string) (body, updatedAt, source string) {
	latestBody, latestAt := latestCommentForLogin(nodes, wantLogin)
	var candidates []prCommentNode
	for _, n := range nodes {
		if n.Author == nil {
			continue
		}
		if normalizeGitHubActorLogin(n.Author.Login) != normalizeGitHubActorLogin(wantLogin) {
			continue
		}
		if commentMaxTier(n.Body, enrichNames) > 0 {
			candidates = append(candidates, n)
		}
	}
	if len(candidates) == 0 {
		return latestBody, latestAt, "fallback_latest"
	}
	maxTier := 0
	for _, n := range candidates {
		if t := commentMaxTier(n.Body, enrichNames); t > maxTier {
			maxTier = t
		}
	}
	var best prCommentNode
	var bestSet bool
	for _, n := range candidates {
		if commentMaxTier(n.Body, enrichNames) != maxTier {
			continue
		}
		if !bestSet || n.UpdatedAt >= best.UpdatedAt {
			best = n
			bestSet = true
		}
	}
	if !bestSet {
		return latestBody, latestAt, "fallback_latest"
	}
	return best.Body, best.UpdatedAt, "status_marker"
}

func headRefOIDFromPR(pr *prPullRequestNode) string {
	if pr == nil || pr.HeadRefOid == nil {
		return ""
	}
	return strings.TrimSpace(*pr.HeadRefOid)
}

// buildBotReviewerStatus builds one status map per configured bot: latest review, issue-comment
// enrichment, finding-conversation counts, rate-limit age adjustment, and ClassifyBotStatus.
func buildBotReviewerStatus(pr *prPullRequestNode, bots []BotReviewer, commentNodes []prCommentNode, now time.Time) []any {
	if len(bots) == 0 {
		return []any{}
	}
	headSHA := headRefOIDFromPR(pr)
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
		statusBody, statusAt, statusSrc := selectStatusCommentForLogin(nodes, login, br.Enrich)
		_, latestAt := latestCommentForLogin(nodes, login)
		statusLower := strings.ToLower(statusBody)
		status := map[string]any{
			"id":                     strings.TrimSpace(br.ID),
			"login":                  login,
			"required":               br.Required,
			"has_review":             hasRev,
			"review_state":           state,
			"review_commit_sha":      ri.CommitOid,
			"latest_comment_at":      latestAt,
			"status_comment_at":      statusAt,
			"status_comment_source":  statusSrc,
			"contains_rate_limit":    strings.Contains(statusLower, "rate limit"),
			"contains_review_paused": strings.Contains(statusLower, "review paused") || strings.Contains(statusLower, "review pause"),
			"contains_review_failed": reviewFailedFromComment(statusLower),
		}
		if extra := applyEnrichments(statusBody, br.Enrich); len(extra) > 0 {
			for k, v := range extra {
				status[k] = v
			}
		}
		if extra := applyReviewBodyEnrichments(ri.Body, br.Enrich); len(extra) > 0 {
			for k, v := range extra {
				status[k] = v
			}
		}
		applyFindingConversationCountIfNeeded(status, br.Enrich, nodes, login)
		adjustRateLimitRemainingForStatusCommentAge(status, now)
		applyReviewCommitSHAFromEnrichedComment(status, headSHA)
		status["status"] = ClassifyBotStatus(status, br.Enrich)
		applyCoderabbitStatusClassBasis(status, br.Enrich)
		out = append(out, status)
	}
	return out
}

// latestCommentForLogin returns the body and updatedAt of the chronologically latest issue comment
// by wantLogin (RFC3339 string compare on updatedAt).
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
