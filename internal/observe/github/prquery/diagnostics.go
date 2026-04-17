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

// IsBlockedTier reports whether s is a non-terminal blocked bot review status.
func IsBlockedTier(s string) bool {
	switch s {
	case BotStatusRateLimited, BotStatusReviewPaused:
		return true
	default:
		return false
	}
}

// IsFailedTier reports whether s is a terminal-failure bot review status.
func IsFailedTier(s string) bool {
	switch s {
	case BotStatusReviewFailed:
		return true
	default:
		return false
	}
}

func blockedReasonForStatus(s string) string {
	if !IsBlockedTier(s) {
		return ""
	}
	return s
}

func combineBlockedReason(current, next string) string {
	if next == "" || next == current {
		return current
	}
	if current == "" {
		return next
	}
	return "mixed"
}

// duplicateFindingsDetected reports whether any required bot reviewer has a positive
// duplicate-finding count (CodeRabbit-enriched field names included).
func duplicateFindingsDetected(statusList []any) bool {
	for _, elt := range statusList {
		m, ok := elt.(map[string]any)
		if !ok {
			continue
		}
		req, ok := m["required"].(bool)
		if !ok || !req {
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

// nonThreadFindingsPresentForStatus is true when any non-thread finding signal on this
// bot's status map is positive (actionable, outside-diff, duplicate, or finding-shaped
// conversation comments).
func nonThreadFindingsPresentForStatus(m map[string]any) bool {
	a := intFromStatusMapOrZeroAny(m, "actionable_findings_count", "cr_actionable_comments_count")
	o := intFromStatusMapOrZeroAny(m, "outside_diff_findings_count", "cr_outside_diff_comments_count")
	d := intFromStatusMapOrZeroAny(m, "duplicate_findings_count", "cr_duplicate_findings_count")
	f := intFromStatusMapOrZeroAny(m, "finding_conversation_comments_count", "cr_finding_conversation_comments_count")
	// f is a raw count of bot-authored issue comments classified as finding-shaped (see
	// IsCoderabbitFindingConversationComment). It does not subtract later user disposition replies;
	// closure for those uses the consensus protocol (PR conversation disposition), not this aggregate alone.
	// Duplicate-suppressed inline counts from the review summary alone (zero actionable/outside) are
	// informational; duplicate_findings_detected still surfaces them for observation.
	if a == 0 && o == 0 && d > 0 {
		d = 0
	}
	return a > 0 || o > 0 || d > 0 || f > 0
}

// hasRequiredBot returns true if statusList contains at least one entry with required: true.
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

// aggregateNonThreadFindings combines per-bot non-thread signals. When conversation
// comments were not fully paginated, it fails closed: true if any required bot exists
// even when counts may be partial.
func aggregateNonThreadFindings(statusList []any, conversationCommentsIncomplete bool) bool {
	n := nonThreadFindingsPresentForRequiredBots(statusList)
	if conversationCommentsIncomplete {
		return n || hasRequiredBot(statusList)
	}
	return n
}

type botReviewAggregate struct {
	blockedReason string
	sawRequired   bool
	anyBlocked    bool
	anyFailed     bool
	allReviewed   bool
	anyStale      bool
}

func newBotReviewAggregate() botReviewAggregate {
	return botReviewAggregate{allReviewed: true}
}

func accumulateRequiredBotReviewState(agg *botReviewAggregate, m map[string]any, headSHA string) {
	req, ok := m["required"].(bool)
	if !ok || !req {
		return
	}
	agg.sawRequired = true
	st, _ := m["status"].(string)
	switch {
	case IsBlockedTier(st):
		agg.anyBlocked = true
		agg.allReviewed = false
		agg.blockedReason = combineBlockedReason(agg.blockedReason, blockedReasonForStatus(st))
	case IsFailedTier(st):
		agg.anyFailed = true
		agg.allReviewed = false
	case IsReviewedTier(st):
		reviewSHA, _ := m["review_commit_sha"].(string)
		if headSHA != "" && (reviewSHA == "" || reviewSHA != headSHA) {
			agg.anyStale = true
		}
	default:
		agg.allReviewed = false
	}
}

func buildBotReviewDiagnostics(agg botReviewAggregate, duplicateFindings, nonThreadFindings bool) map[string]any {
	if !agg.sawRequired {
		return map[string]any{
			"bot_review_completed":        true,
			"bot_review_pending":          false,
			"bot_review_terminal":         true,
			"bot_review_blocked":          false,
			"bot_review_block_reason":     "",
			"bot_review_failed":           false,
			"bot_review_stale":            false,
			"duplicate_findings_detected": duplicateFindings,
			"non_thread_findings_present": nonThreadFindings,
		}
	}

	completed := !agg.anyFailed && !agg.anyBlocked && agg.allReviewed
	terminal := agg.anyFailed || completed
	pending := !terminal && !agg.anyBlocked

	return map[string]any{
		"bot_review_completed":        completed,
		"bot_review_pending":          pending,
		"bot_review_terminal":         terminal,
		"bot_review_blocked":          agg.anyBlocked,
		"bot_review_block_reason":     agg.blockedReason,
		"bot_review_failed":           agg.anyFailed,
		"bot_review_stale":            agg.anyStale,
		"duplicate_findings_detected": duplicateFindings,
		"non_thread_findings_present": nonThreadFindings,
	}
}

// intFromStatusMapOrZeroAny returns the first matching key's int value, or 0.
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
	duplicateFindings := duplicateFindingsDetected(statusList)
	nonThreadFindings := aggregateNonThreadFindings(statusList, conversationCommentsIncomplete)
	agg := newBotReviewAggregate()

	for _, elt := range statusList {
		m, ok := elt.(map[string]any)
		if !ok {
			continue
		}
		accumulateRequiredBotReviewState(&agg, m, headSHA)
	}

	return buildBotReviewDiagnostics(agg, duplicateFindings, nonThreadFindings)
}

// intFromStatusMapAny returns the first present key among keys that holds an int-like value.
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
