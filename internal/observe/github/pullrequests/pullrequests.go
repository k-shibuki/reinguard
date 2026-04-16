// Package pullrequests implements PR-related signals (open count and current-branch context)
// for the observe GitHub provider (ADR-0006). Collect returns maps and warnings or an error on failure.
package pullrequests

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/k-shibuki/reinguard/internal/githubapi"
	"github.com/k-shibuki/reinguard/internal/gitroot"
)

// prSearchItem is a subset of GitHub issue search results (open PR count).
type prSearchItem struct {
	Number int `json:"number"`
}

// searchResponse is GitHub issue-search JSON for the open PR count query.
type searchResponse struct {
	Items      []prSearchItem `json:"items"`
	TotalCount int            `json:"total_count"`
}

// pullHead is the head branch ref on a pull request list item.
type pullHead struct {
	Repo *struct {
		Owner *struct {
			Login string `json:"login"`
		} `json:"owner"`
		Name string `json:"name"`
	} `json:"repo"`
	Ref string `json:"ref"`
	SHA string `json:"sha"`
}

type pullBase struct {
	Ref string `json:"ref"`
}

type pullLabel struct {
	Name string `json:"name"`
}

type pullGet struct {
	Mergeable      *bool       `json:"mergeable"`
	Head           pullHead    `json:"head"`
	Base           pullBase    `json:"base"`
	MergeableState string      `json:"mergeable_state"`
	State          string      `json:"state"`
	Title          string      `json:"title"`
	Labels         []pullLabel `json:"labels"`
	Number         int         `json:"number"`
	Draft          bool        `json:"draft"`
}

// pullListItem is one element from GET /repos/{owner}/{repo}/pulls for branch matching.
type pullListItem struct {
	Head   pullHead `json:"head"`
	Number int      `json:"number"`
}

// ScopeOptions configures explicit PR or branch selection for pull-request observation.
type ScopeOptions struct {
	Branch   string
	PRNumber int
}

// Scope describes which branch / PR the pull-request facet observed.
type Scope struct {
	LocalBranch       string
	RequestedBranch   string
	EffectiveBranch   string
	Selection         string
	RequestedPRNumber int
	ResolvedPRNumber  int
}

// Selection values for Scope.Selection.
const (
	SelectionExplicitPR     = "explicit_pr"
	SelectionExplicitBranch = "explicit_branch"
	SelectionCurrentBranch  = "current_branch"
	SelectionNone           = "none"

	// ViewSummary requests the summary-only pull-request output contract.
	ViewSummary = "summary"
)

// Collect returns pull request signals for the effective observed branch / PR.
func Collect(ctx context.Context, c *githubapi.Client, owner, repo, workDir string, opts ScopeOptions, view string) (map[string]any, Scope, []string, error) {
	if c == nil {
		return nil, Scope{}, nil, fmt.Errorf("nil client")
	}
	var warnings []string
	localBranch, w := resolveBranch(ctx, workDir)
	warnings = append(warnings, w...)
	scope := Scope{
		LocalBranch:       localBranch,
		RequestedBranch:   strings.TrimSpace(opts.Branch),
		RequestedPRNumber: opts.PRNumber,
	}

	qOpen := fmt.Sprintf("repo:%s/%s is:pr is:open", owner, repo)
	uOpen := c.APIBase() + "/search/issues?q=" + url.QueryEscape(qOpen)
	var openOut searchResponse
	if err := c.GetJSON(ctx, uOpen, &openOut); err != nil {
		return nil, scope, warnings, err
	}

	branch := scope.RequestedBranch
	if branch == "" {
		branch = localBranch
	}
	scope.EffectiveBranch = branch
	prForBranch, prNum, explicitPull, err := resolvePRSelection(ctx, c, owner, repo, branch, &scope)
	if err != nil {
		return nil, scope, warnings, err
	}
	scope.ResolvedPRNumber = prNum

	prSignals := map[string]any{
		"open_count":           openOut.TotalCount,
		"current_branch":       scope.EffectiveBranch,
		"pr_exists_for_branch": prForBranch,
		"pr_number_for_branch": prNum,
		"observed_scope":       scope.signalMap(),
	}
	if prForBranch && strings.EqualFold(strings.TrimSpace(view), ViewSummary) {
		detail, err := summaryPullDetail(ctx, c, owner, repo, prNum, explicitPull)
		if err != nil {
			return nil, scope, warnings, err
		}
		mergePullSummary(prSignals, detail)
	}

	return map[string]any{
		"pull_requests": prSignals,
	}, scope, warnings, nil
}

func resolvePRSelection(ctx context.Context, c *githubapi.Client, owner, repo, branch string, scope *Scope) (prForBranch bool, prNum int, explicitPull *pullGet, err error) {
	switch {
	case scope.RequestedPRNumber > 0:
		scope.Selection = SelectionExplicitPR
		pull, err := fetchPullRequest(ctx, c, owner, repo, scope.RequestedPRNumber)
		if err != nil {
			return false, 0, nil, err
		}
		if !strings.EqualFold(strings.TrimSpace(pull.State), "open") {
			return false, 0, nil, fmt.Errorf("pull request #%d is not open", scope.RequestedPRNumber)
		}
		scope.EffectiveBranch = strings.TrimSpace(pull.Head.Ref)
		return true, pull.Number, &pull, nil
	case branch != "":
		scope.Selection = SelectionCurrentBranch
		if scope.RequestedBranch != "" {
			scope.Selection = SelectionExplicitBranch
		}
		// Issue search `head:<branch>` matches by prefix; use List Pulls with
		// head=owner:branch for an exact head ref (GitHub REST).
		q := url.Values{}
		q.Set("state", "open")
		q.Set("head", owner+":"+branch)
		uPulls := fmt.Sprintf("%s/repos/%s/%s/pulls?%s",
			c.APIBase(),
			url.PathEscape(owner),
			url.PathEscape(repo),
			q.Encode(),
		)
		var pulls []pullListItem
		if err := c.GetJSON(ctx, uPulls, &pulls); err != nil {
			return false, 0, nil, err
		}
		for _, p := range pulls {
			if strings.EqualFold(p.Head.Ref, branch) {
				return true, p.Number, nil, nil
			}
		}
		return false, 0, nil, nil
	default:
		scope.Selection = SelectionNone
		return false, 0, nil, nil
	}
}

func summaryPullDetail(ctx context.Context, c *githubapi.Client, owner, repo string, prNum int, explicitPull *pullGet) (pullGet, error) {
	if explicitPull != nil {
		return *explicitPull, nil
	}
	return fetchPullRequest(ctx, c, owner, repo, prNum)
}

// resolveBranch returns the current branch name or warnings for detached HEAD / errors.
func resolveBranch(ctx context.Context, workDir string) (branch string, warnings []string) {
	b, detached, err := gitroot.CurrentBranch(ctx, workDir)
	if err != nil {
		return "", []string{err.Error()}
	}
	if detached {
		return "", []string{"detached HEAD"}
	}
	return b, nil
}

func fetchPullRequest(ctx context.Context, c *githubapi.Client, owner, repo string, number int) (pullGet, error) {
	u := fmt.Sprintf("%s/repos/%s/%s/pulls/%d",
		c.APIBase(),
		url.PathEscape(owner),
		url.PathEscape(repo),
		number,
	)
	var pull pullGet
	if err := c.GetJSON(ctx, u, &pull); err != nil {
		return pullGet{}, err
	}
	return pull, nil
}

func (s Scope) signalMap() map[string]any {
	out := map[string]any{
		"local_branch_at_collect": s.LocalBranch,
		"selection":               s.Selection,
		"resolved_pr_number":      s.ResolvedPRNumber,
	}
	if s.RequestedBranch != "" {
		out["requested_branch"] = s.RequestedBranch
	}
	if s.RequestedPRNumber > 0 {
		out["requested_pr_number"] = s.RequestedPRNumber
	}
	if s.EffectiveBranch != "" {
		out["effective_branch"] = s.EffectiveBranch
	}
	return out
}

// mergePullSummary populates dst with normalized PR detail fields.
// It overwrites current_branch with the authoritative head_ref from the PR.
func mergePullSummary(dst map[string]any, pull pullGet) {
	dst["state"] = strings.ToLower(strings.TrimSpace(pull.State))
	dst["draft"] = pull.Draft
	if title := strings.TrimSpace(pull.Title); title != "" {
		dst["title"] = title
	}
	if baseRef := strings.TrimSpace(pull.Base.Ref); baseRef != "" {
		dst["base_ref"] = baseRef
	}
	if headRef := strings.TrimSpace(pull.Head.Ref); headRef != "" {
		dst["head_ref"] = headRef
		dst["current_branch"] = headRef
	}
	if headSHA := strings.TrimSpace(pull.Head.SHA); headSHA != "" {
		dst["head_sha"] = headSHA
	}
	if pull.Head.Repo != nil {
		if owner := pull.Head.Repo.Owner; owner != nil && strings.TrimSpace(owner.Login) != "" {
			dst["head_repo_owner"] = strings.TrimSpace(owner.Login)
		}
		if repoName := strings.TrimSpace(pull.Head.Repo.Name); repoName != "" {
			dst["head_repo_name"] = repoName
		}
	}
	switch {
	case pull.Mergeable == nil:
		dst["mergeable"] = "unknown"
	case *pull.Mergeable:
		dst["mergeable"] = "mergeable"
	default:
		dst["mergeable"] = "conflicting"
	}
	if mergeState := strings.TrimSpace(pull.MergeableState); mergeState != "" {
		dst["merge_state_status"] = strings.ToLower(mergeState)
	}
	labels := make([]any, 0, len(pull.Labels))
	for _, label := range pull.Labels {
		if name := strings.TrimSpace(label.Name); name != "" {
			labels = append(labels, name)
		}
	}
	dst["labels"] = labels
}
