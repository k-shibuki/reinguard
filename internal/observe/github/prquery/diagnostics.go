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

// ComputeBotReviewDiagnostics aggregates per-bot status for required reviewers.
// When no required bots are configured, all flags reflect vacuous completion (nothing to wait on).
func ComputeBotReviewDiagnostics(statusList []any) map[string]any {
	var sawRequired bool
	anyFailed := false
	allReviewed := true

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
	}

	if !sawRequired {
		return map[string]any{
			"bot_review_completed": true,
			"bot_review_pending":   false,
			"bot_review_terminal":  true,
			"bot_review_failed":    false,
		}
	}

	completed := !anyFailed && allReviewed
	terminal := anyFailed || completed
	pending := !terminal

	return map[string]any{
		"bot_review_completed": completed,
		"bot_review_pending":   pending,
		"bot_review_terminal":  terminal,
		"bot_review_failed":    anyFailed,
	}
}
