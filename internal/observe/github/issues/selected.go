package issues

import (
	"context"
	"fmt"
	"strings"

	"github.com/k-shibuki/reinguard/internal/githubapi"
)

type restLabel struct {
	Name string `json:"name"`
}

type restMilestone struct {
	Title string `json:"title"`
}

type restAssignee struct {
	Login string `json:"login"`
}

type restIssue struct { //nolint:govet // fieldalignment: JSON field order matches GitHub API
	Number    int            `json:"number"`
	State     string         `json:"state"`
	Title     string         `json:"title"`
	Labels    []restLabel    `json:"labels"`
	Milestone *restMilestone `json:"milestone"`
	Assignees []restAssignee `json:"assignees"`
	Body      string         `json:"body"`
}

// CollectSelected fetches each issue by number via REST and returns a JSON-ready []any slice.
// On failure for one issue when len(numbers) > 1, appends {"number": n, "error": "..."}.
// When len(numbers) == 1 and that issue is not found (404), returns (nil, err) wrapping ErrFatalObservation.
func CollectSelected(ctx context.Context, c *githubapi.Client, owner, repo string, numbers []int) ([]any, error) {
	if c == nil {
		return nil, fmt.Errorf("issues: nil client")
	}
	owner = strings.TrimSpace(owner)
	repo = strings.TrimSpace(repo)
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("issues: owner and repo must be non-empty")
	}
	if len(numbers) == 0 {
		return nil, nil
	}

	out := make([]any, 0, len(numbers))
	single := len(numbers) == 1

	for _, num := range numbers {
		if num <= 0 {
			out = append(out, map[string]any{"number": num, "error": "invalid issue number"})
			continue
		}
		u := fmt.Sprintf("%s/repos/%s/%s/issues/%d", c.APIBase(), owner, repo, num)
		var ri restIssue
		err := c.GetJSON(ctx, u, &ri)
		if err != nil {
			if single && isNotFound(err) {
				return nil, fmt.Errorf("issues: issue #%d not found: %w", num, ErrFatalObservation)
			}
			out = append(out, map[string]any{"number": num, "error": err.Error()})
			continue
		}
		out = append(out, issueToMap(ri))
	}
	return out, nil
}

func isNotFound(err error) bool {
	if err == nil {
		return false
	}
	s := err.Error()
	return strings.Contains(s, " 404 ") || strings.Contains(s, ": 404 ")
}

func issueToMap(ri restIssue) map[string]any {
	labelNames := make([]string, 0, len(ri.Labels))
	for _, l := range ri.Labels {
		if strings.TrimSpace(l.Name) != "" {
			labelNames = append(labelNames, l.Name)
		}
	}
	isEpic := false
	for _, n := range labelNames {
		if strings.EqualFold(n, "epic") {
			isEpic = true
			break
		}
	}
	assignees := make([]string, 0, len(ri.Assignees))
	for _, a := range ri.Assignees {
		if strings.TrimSpace(a.Login) != "" {
			assignees = append(assignees, a.Login)
		}
	}
	var milestone any
	if ri.Milestone != nil && strings.TrimSpace(ri.Milestone.Title) != "" {
		milestone = ri.Milestone.Title
	} else {
		milestone = nil
	}
	return map[string]any{
		"number":        ri.Number,
		"state":         strings.ToLower(strings.TrimSpace(ri.State)),
		"title":         ri.Title,
		"labels":        labelNames,
		"has_blockers":  HasBlockers(ri.Body),
		"is_epic":       isEpic,
		"milestone":     milestone,
		"assignees":     assignees,
		"body_sections": ParseSections(ri.Body),
	}
}
