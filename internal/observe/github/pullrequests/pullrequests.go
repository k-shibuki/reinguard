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

type searchResponse struct {
	Items      []prSearchItem `json:"items"`
	TotalCount int            `json:"total_count"`
}

type pullHead struct {
	Ref string `json:"ref"`
}

type pullListItem struct {
	Head   pullHead `json:"head"`
	Number int      `json:"number"`
}

// Collect returns pull request signals for the current branch.
func Collect(ctx context.Context, c *githubapi.Client, owner, repo, workDir string) (map[string]any, []string, error) {
	if c == nil {
		return nil, nil, fmt.Errorf("nil client")
	}
	var warnings []string
	branch, w := resolveBranch(ctx, workDir)
	warnings = append(warnings, w...)

	qOpen := fmt.Sprintf("repo:%s/%s is:pr is:open", owner, repo)
	uOpen := c.APIBase() + "/search/issues?q=" + url.QueryEscape(qOpen)
	var openOut searchResponse
	if err := c.GetJSON(ctx, uOpen, &openOut); err != nil {
		return nil, warnings, err
	}

	prForBranch := false
	prNum := 0
	if branch != "" {
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
			return nil, warnings, err
		}
		for _, p := range pulls {
			if strings.EqualFold(p.Head.Ref, branch) {
				prForBranch = true
				prNum = p.Number
				break
			}
		}
	}

	return map[string]any{
		"pull_requests": map[string]any{
			"open_count":           openOut.TotalCount,
			"current_branch":       branch,
			"pr_exists_for_branch": prForBranch,
			"pr_number_for_branch": prNum,
		},
	}, warnings, nil
}

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
