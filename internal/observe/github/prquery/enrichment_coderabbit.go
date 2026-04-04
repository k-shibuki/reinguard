package prquery

import (
	"regexp"
	"strconv"
	"strings"
)

type coderabbitEnrichment struct{}

func (coderabbitEnrichment) Name() string { return "coderabbit" }

// Enrich extracts rate-limit timing and CodeRabbit Review Status markers from issue comments.
func (coderabbitEnrichment) Enrich(commentBody string) map[string]any {
	body := strings.TrimSpace(commentBody)
	if body == "" {
		return nil
	}
	out := make(map[string]any)
	if sec := parseCoderabbitRateLimitSeconds(body); sec > 0 {
		out["rate_limit_remaining_seconds"] = sec
	}
	for k, v := range parseCoderabbitReviewStatusMarkers(body) {
		out[k] = v
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// EnrichReviewBody parses CodeRabbit PullRequestReview.summary bodies (e.g. duplicate-comment notices).
func (coderabbitEnrichment) EnrichReviewBody(reviewBody string) map[string]any {
	n := parseCoderabbitDuplicateCount(reviewBody)
	if n <= 0 {
		return nil
	}
	return map[string]any{"cr_duplicate_findings_count": n}
}

// parseCoderabbitDuplicateCount extracts N from a "♻️ Duplicate comments (N)" line in a review body.
func parseCoderabbitDuplicateCount(body string) int {
	if m := reCRDuplicateComments.FindStringSubmatch(body); len(m) >= 2 {
		n, err := strconv.Atoi(m[1])
		if err != nil || n < 0 {
			return 0
		}
		return n
	}
	return 0
}

// ClassifyStatus applies CodeRabbit-aware rules on top of generic substring flags.
func (coderabbitEnrichment) ClassifyStatus(m map[string]any) string {
	if signalBool(m, "contains_rate_limit") {
		return BotStatusRateLimited
	}
	if signalBool(m, "contains_review_paused") {
		return BotStatusReviewPaused
	}
	if signalBool(m, "contains_review_failed") {
		return BotStatusReviewFailed
	}
	if v, ok := m["cr_review_processing"].(bool); ok && v {
		return BotStatusPending
	}
	// Terminal clean without a GitHub Review (issue-comment-only completion, e.g. "no actionable comments").
	if signalBool(m, "cr_review_completed_clean") {
		return BotStatusCompletedClean
	}
	if signalBool(m, "has_review") {
		return BotStatusCompleted
	}
	if signalBool(m, "cr_walkthrough_present") {
		return BotStatusPending
	}
	if strings.TrimSpace(signalString(m, "latest_comment_at")) != "" {
		return BotStatusPending
	}
	return BotStatusNotTriggered
}

// Heuristic markers for CodeRabbit "Review Status" / walkthrough issue comments (best-effort).
var (
	// U+267B recycling symbol, optional U+FE0F variation selector (emoji presentation).
	reCRDuplicateComments = regexp.MustCompile(`(?i)(?:\x{267B}\x{FE0F}?|\x{267B})\s*Duplicate\s+comments\s*\((\d+)\)`)
	reCRReviewCompleted   = regexp.MustCompile(`(?i)(review\s+completed|✅\s*completed|status[:\s\*]*\s*✅|\*\*status\*\*[^\n]*(complete|finished|done))`)
	reCRReviewClean       = regexp.MustCompile(`(?i)(no\s+issues\s+found|no\s+additional\s+issues\s+found|no\s+actionable\s+comments)`)
	// Second bracketed SHA is the reviewed head when CodeRabbit summarizes a commit range in PR comments.
	reCRReviewedHeadRange = regexp.MustCompile(`(?i)between\s+\[([0-9a-f]{7,40})\]\([^)]*\)\s+and\s+\[([0-9a-f]{7,40})\]\(`)
	reCRReviewProcessing  = regexp.MustCompile(`(?i)(processing\s+new\s+changes|review\s+in\s+progress|\bin\s+progress\b|\*\*status\*\*[^\n]*(in\s+progress|pending|processing)|⏳\s*processing)`)
	reCRWalkthrough       = regexp.MustCompile(`(?i)(^|\n)#{1,3}\s*[^\n]*(walkthrough|review\s+walkthrough)|\*\*walkthrough\*\*`)
)

func parseCoderabbitReviewedHeadSHA(body string) string {
	if m := reCRReviewedHeadRange.FindStringSubmatch(body); len(m) >= 3 {
		return strings.ToLower(strings.TrimSpace(m[2]))
	}
	return ""
}

func parseCoderabbitReviewStatusMarkers(body string) map[string]any {
	out := make(map[string]any)
	if sha := parseCoderabbitReviewedHeadSHA(body); sha != "" {
		out["cr_reviewed_head_sha"] = sha
	}
	if reCRWalkthrough.MatchString(body) {
		out["cr_walkthrough_present"] = true
	}
	// "No issues found" is a terminal clean result; other status markers do not add value after this.
	if reCRReviewClean.MatchString(body) {
		out["cr_review_processing"] = false
		out["cr_review_completed_clean"] = true
		return out
	}
	if reCRReviewCompleted.MatchString(body) {
		out["cr_review_processing"] = false
		return out
	}
	if reCRReviewProcessing.MatchString(body) {
		out["cr_review_processing"] = true
	}
	return out
}

// Matches: "try again in 5 minutes and 30 seconds", "try again in 1 minute", "in 2 minutes and 15 seconds"
var (
	reTryAgainMinutesSeconds = regexp.MustCompile(`(?i)try\s+again\s+in\s+(\d+)\s*minutes?(?:\s+and\s+(\d+)\s*seconds?)?`)
	reInMinutesSeconds       = regexp.MustCompile(`(?i)in\s+(\d+)\s*minutes?(?:\s+and\s+(\d+)\s*seconds?)?`)
	reSecondsOnly            = regexp.MustCompile(`(?i)(?:try\s+again\s+)?in\s+(\d+)\s*seconds?`)
)

func parseCoderabbitRateLimitSeconds(body string) int {
	body = strings.TrimSpace(body)
	if body == "" {
		return 0
	}
	if m := reTryAgainMinutesSeconds.FindStringSubmatch(body); len(m) >= 2 {
		return minutesSecondsToTotal(m[1], subOrEmpty(m, 2))
	}
	if m := reInMinutesSeconds.FindStringSubmatch(body); len(m) >= 2 {
		return minutesSecondsToTotal(m[1], subOrEmpty(m, 2))
	}
	if m := reSecondsOnly.FindStringSubmatch(body); len(m) >= 2 {
		if s, err := strconv.Atoi(m[1]); err == nil && s >= 0 {
			return s
		}
	}
	return 0
}

func subOrEmpty(m []string, i int) string {
	if i < len(m) && m[i] != "" {
		return m[i]
	}
	return ""
}

func minutesSecondsToTotal(minStr, secStr string) int {
	minutes, err := strconv.Atoi(minStr)
	if err != nil || minutes < 0 {
		return 0
	}
	total := minutes * 60
	if secStr != "" {
		s, err := strconv.Atoi(secStr)
		if err == nil && s >= 0 {
			total += s
		}
	}
	return total
}
