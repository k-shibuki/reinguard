package pullrequests

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"

	"github.com/k-shibuki/reinguard/internal/githubapi"
)

type pull struct {
	State string `json:"state"`
	Head  struct {
		Ref string `json:"ref"`
	} `json:"head"`
	Number int `json:"number"`
}

// Collect returns pull request signals for the current branch.
func Collect(ctx context.Context, c *githubapi.Client, owner, repo, workDir string) (map[string]any, []string, error) {
	if c == nil {
		return nil, nil, fmt.Errorf("nil client")
	}
	var warnings []string
	branch, err := currentBranch(ctx, workDir)
	if err != nil {
		warnings = append(warnings, err.Error())
		branch = ""
	}
	u := fmt.Sprintf("%s/repos/%s/%s/pulls?state=open&per_page=100", c.APIBase(), owner, repo)
	var list []pull
	if err := c.GetJSON(ctx, u, &list); err != nil {
		return nil, warnings, err
	}
	prForBranch := false
	prNum := 0
	for _, p := range list {
		if branch != "" && strings.EqualFold(p.Head.Ref, branch) {
			prForBranch = true
			prNum = p.Number
			break
		}
	}
	return map[string]any{
		"pull_requests": map[string]any{
			"open_count":           len(list),
			"current_branch":       branch,
			"pr_exists_for_branch": prForBranch,
			"pr_number_for_branch": prNum,
		},
	}, warnings, nil
}

func currentBranch(ctx context.Context, wd string) (string, error) {
	cmd := exec.CommandContext(ctx, "git", "rev-parse", "--abbrev-ref", "HEAD")
	if wd != "" {
		cmd.Dir = wd
	}
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git branch: %w", err)
	}
	b := strings.TrimSpace(buf.String())
	if b == "HEAD" {
		return "", fmt.Errorf("detached HEAD")
	}
	return b, nil
}
