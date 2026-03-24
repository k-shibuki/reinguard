package pullrequests

import (
	"context"
	"fmt"
	"net/url"
	"strings"

	"github.com/k-shibuki/reinguard/internal/githubapi"
	"github.com/k-shibuki/reinguard/internal/gitroot"
)

type prSearchHead struct {
	Ref string `json:"ref"`
}

type prSearchItem struct {
	Head   prSearchHead `json:"head"`
	Number int          `json:"number"`
}

type searchResponse struct {
	Items      []prSearchItem `json:"items"`
	TotalCount int            `json:"total_count"`
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
		qh := fmt.Sprintf("repo:%s/%s is:pr is:open head:%s", owner, repo, branch)
		uh := c.APIBase() + "/search/issues?q=" + url.QueryEscape(qh)
		var headOut searchResponse
		if err := c.GetJSON(ctx, uh, &headOut); err != nil {
			return nil, warnings, err
		}
		for _, it := range headOut.Items {
			if strings.EqualFold(it.Head.Ref, branch) {
				prForBranch = true
				prNum = it.Number
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
