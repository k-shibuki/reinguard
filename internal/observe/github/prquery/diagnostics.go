package prquery

// Bot review status tier values (observation + FSM). See ADR-0013 / docs/cli.md.
const (
	BotStatusNotTriggered   = "not_triggered"
	BotStatusPending        = "pending"
	BotStatusCompleted      = "completed"
	BotStatusCompletedClean = "completed_clean"
	BotStatusRateLimited    = "rate_limited"
	BotStatusReviewPaused   = "review_paused"
	BotStatusReviewFailed   = "review_failed"
)

// IsReviewedTier reports whether s is a terminal-success bot review status.
func IsReviewedTier(s string) bool {
	switch s {
	case BotStatusCompleted, BotStatusCompletedClean:
		return true
	default:
		return false
	}
}

// IsFailedTier reports whether s is a terminal-failure bot review status.
func IsFailedTier(s string) bool {
	switch s {
	case BotStatusRateLimited, BotStatusReviewPaused, BotStatusReviewFailed:
		return true
	default:
		return false
	}
}

func duplicateFindingsDetected(statusList []any) bool {
	for _, elt := range statusList {
		m, ok := elt.(map[string]any)
		if !ok {
			continue
		}
		if n, ok := intFromStatusMapAny(m, "duplicate_findings_count", "cr_duplicate_findings_count"); ok && n > 0 {
			return true
		}
	}
	return false
}

// nonThreadFindingsPresentForRequiredBots is true when any required bot has a positive
// actionable, outside-diff, duplicate, or finding-conversation count (Issue #105).
func nonThreadFindingsPresentForRequiredBots(statusList []any) bool {
	for _, elt := range statusList {
		m, ok := elt.(map[string]any)
		if !ok {
			continue
		}
		req, ok := m["required"].(bool)
		if !ok || !req {
			continue
		}
		if nonThreadFindingsPresentForStatus(m) {
			return true
		}
	}
	return false
}

func nonThreadFindingsPresentForStatus(m map[string]any) bool {
	a := intFromStatusMapOrZeroAny(m, "actionable_findings_count", "cr_actionable_comments_count")
	o := intFromStatusMapOrZeroAny(m, "outside_diff_findings_count", "cr_outside_diff_comments_count")
	d := intFromStatusMapOrZeroAny(m, "duplicate_findings_count", "cr_duplicate_findings_count")
	f := intFromStatusMapOrZeroAny(m, "finding_conversation_comments_count", "cr_finding_conversation_comments_count")
	// f is a raw count of bot-authored issue comments classified as finding-shaped (see
	// IsCoderabbitFindingConversationComment). It does not subtract later human disposition replies;
	// closure for those uses the consensus protocol (PR conversation disposition), not this aggregate alone.
	return a > 0 || o > 0 || d > 0 || f > 0
}

func hasRequiredBot(statusList []any) bool {
	for _, elt := range statusList {
		m, ok := elt.(map[string]any)
		if !ok {
			continue
		}
		if req, ok := m["required"].(bool); ok && req {
			return true
		}
	}
	return false
}

func aggregateNonThreadFindings(statusList []any, conversationCommentsIncomplete bool) bool {
	n := nonThreadFindingsPresentForRequiredBots(statusList)
	if conversationCommentsIncomplete {
		return n || hasRequiredBot(statusList)
	}
	return n
}

func intFromStatusMapOrZeroAny(m map[string]any, keys ...string) int {
	n, ok := intFromStatusMapAny(m, keys...)
	if !ok {
		return 0
	}
	return n
}

// ComputeBotReviewDiagnostics aggregates per-bot status for required reviewers.
// headSHA is the PR HEAD commit; when a required bot's review targets a different
// commit, bot_review_stale is true. When no required bots are configured, all
// flags reflect vacuous completion (nothing to wait on).
// When conversationCommentsIncomplete is true, non-thread counts may be partial;
// non_thread_findings_present is fail-closed true if any required bot is configured.
func ComputeBotReviewDiagnostics(statusList []any, headSHA string, conversationCommentsIncomplete bool) map[string]any {
	var sawRequired bool
	anyFailed := false
	allReviewed := true
	anyStale := false
	duplicateFindings := duplicateFindingsDetected(statusList)
	nonThreadFindings := aggregateNonThreadFindings(statusList, conversationCommentsIncomplete)

	for _, elt := range statusList {
		m, ok := elt.(map[string]any)
		if !ok {
			continue
		}
		req, ok := m["required"].(bool)
		if !ok || !req {
			continue
		}
		sawRequired = true
		st, _ := m["status"].(string)
		if IsFailedTier(st) {
			anyFailed = true
			allReviewed = false
			continue
		}
		if !IsReviewedTier(st) {
			allReviewed = false
		}
		if IsReviewedTier(st) {
			reviewSHA, _ := m["review_commit_sha"].(string)
			if headSHA != "" && (reviewSHA == "" || reviewSHA != headSHA) {
				anyStale = true
			}
		}
	}

	if !sawRequired {
		return map[string]any{
			"bot_review_completed":        true,
			"bot_review_pending":          false,
			"bot_review_terminal":         true,
			"bot_review_failed":           false,
			"bot_review_stale":            false,
			"duplicate_findings_detected": duplicateFindings,
			"non_thread_findings_present": nonThreadFindings,
		}
	}

	completed := !anyFailed && allReviewed
	terminal := anyFailed || completed
	pending := !terminal

	return map[string]any{
		"bot_review_completed":        completed,
		"bot_review_pending":          pending,
		"bot_review_terminal":         terminal,
		"bot_review_failed":           anyFailed,
		"bot_review_stale":            anyStale,
		"duplicate_findings_detected": duplicateFindings,
		"non_thread_findings_present": nonThreadFindings,
	}
}

func intFromStatusMapAny(m map[string]any, keys ...string) (int, bool) {
	for _, key := range keys {
		v, ok := m[key]
		if !ok {
			continue
		}
		switch n := v.(type) {
		case int:
			return n, true
		case int64:
			return int(n), true
		case float64:
			return int(n), true
		}
	}
	return 0, false
}
