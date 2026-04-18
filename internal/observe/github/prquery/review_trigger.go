package prquery

import (
	"regexp"
	"strings"
)

// selectLatestReviewTrigger returns the updatedAt (RFC3339) of the newest PR
// conversation comment whose body matches any compiled trigger regex. Comments
// authored by the bot login are ignored so human/agent trigger posts are visible.
func selectLatestReviewTrigger(nodes []prCommentNode, botLogin string, triggers []*regexp.Regexp) string {
	if len(triggers) == 0 {
		return ""
	}
	botKey := normalizeGitHubActorLogin(botLogin)
	var bestAt string
	for _, n := range nodes {
		if n.Author == nil {
			continue
		}
		if normalizeGitHubActorLogin(n.Author.Login) == botKey {
			continue
		}
		body := n.Body
		matched := false
		for _, re := range triggers {
			if re != nil && re.MatchString(body) {
				matched = true
				break
			}
		}
		if !matched {
			continue
		}
		if n.UpdatedAt >= bestAt {
			bestAt = n.UpdatedAt
		}
	}
	return bestAt
}

// maxRFC3339String returns the lexicographically greater non-empty RFC3339
// timestamp, or the other if one is empty. GitHub API timestamps sort as strings
// when formatted consistently (same as latestCommentForLogin tie-breaks).
func maxRFC3339String(a, b string) string {
	a = strings.TrimSpace(a)
	b = strings.TrimSpace(b)
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	if a >= b {
		return a
	}
	return b
}

// computeReviewTriggerAwaitingAck is true when a matching trigger comment is
// newer than the freshest observable bot acknowledgement (status comment time
// and/or review submission time).
func computeReviewTriggerAwaitingAck(triggerAt, statusCommentAt, reviewSubmittedAt string) bool {
	triggerAt = strings.TrimSpace(triggerAt)
	if triggerAt == "" {
		return false
	}
	ack := maxRFC3339String(statusCommentAt, reviewSubmittedAt)
	if ack == "" {
		return true
	}
	return triggerAt > ack
}

func applyReviewTriggerFields(status map[string]any, nodes []prCommentNode, br BotReviewer, statusCommentAt, reviewSubmittedAt string) {
	if len(br.ReviewTriggers) == 0 {
		return
	}
	trAt := selectLatestReviewTrigger(nodes, br.Login, br.ReviewTriggers)
	if trAt != "" {
		status["latest_review_trigger_at"] = trAt
	}
	await := computeReviewTriggerAwaitingAck(trAt, statusCommentAt, reviewSubmittedAt)
	if await {
		status["review_trigger_awaiting_ack"] = true
	}
}
