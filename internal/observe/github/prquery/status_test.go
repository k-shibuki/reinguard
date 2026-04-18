package prquery

import (
	"testing"
)

func assertDiagBool(t *testing.T, got map[string]any, key string, want bool) {
	t.Helper()
	g, ok := got[key].(bool)
	if !ok {
		t.Fatalf("%s: expected bool, got %T in %+v", key, got[key], got)
	}
	if g != want {
		t.Fatalf("%s: got %v, want %v", key, g, want)
	}
}

func assertDiagString(t *testing.T, got map[string]any, key, want string) {
	t.Helper()
	g, ok := got[key].(string)
	if !ok {
		t.Fatalf("%s: expected string, got %T in %+v", key, got[key], got)
	}
	if g != want {
		t.Fatalf("%s: got %q, want %q", key, g, want)
	}
}

func TestComputeBotReviewDiagnostics_vacuousNoRequired(t *testing.T) {
	t.Parallel()
	got := ComputeBotReviewDiagnostics([]any{}, "abc123", false)
	assertDiagBool(t, got, "bot_review_completed", true)
	assertDiagBool(t, got, "bot_review_pending", false)
	assertDiagBool(t, got, "bot_review_terminal", true)
	assertDiagBool(t, got, "bot_review_blocked", false)
	assertDiagString(t, got, "bot_review_block_reason", "")
	assertDiagBool(t, got, "bot_review_failed", false)
	assertDiagBool(t, got, "bot_review_stale", false)
	assertDiagBool(t, got, "duplicate_findings_detected", false)
	assertDiagBool(t, got, "non_thread_findings_present", false)
}

func TestComputeBotReviewDiagnostics_requiredPending(t *testing.T) {
	t.Parallel()
	got := ComputeBotReviewDiagnostics([]any{
		map[string]any{"required": true, "status": BotStatusPending},
	}, "abc123", false)
	assertDiagBool(t, got, "bot_review_completed", false)
	assertDiagBool(t, got, "bot_review_pending", true)
	assertDiagBool(t, got, "bot_review_terminal", false)
	assertDiagBool(t, got, "bot_review_failed", false)
	assertDiagBool(t, got, "bot_review_blocked", false)
}

func TestComputeBotReviewDiagnostics_requiredCompleted(t *testing.T) {
	t.Parallel()
	got := ComputeBotReviewDiagnostics([]any{
		map[string]any{"required": true, "status": BotStatusCompleted, "review_commit_sha": "abc123"},
	}, "abc123", false)
	assertDiagBool(t, got, "bot_review_completed", true)
	assertDiagBool(t, got, "bot_review_pending", false)
	assertDiagBool(t, got, "bot_review_terminal", true)
	assertDiagBool(t, got, "bot_review_failed", false)
	assertDiagBool(t, got, "bot_review_blocked", false)
	assertDiagBool(t, got, "bot_review_stale", false)
}

func TestComputeBotReviewDiagnostics_requiredRateLimitedBlocked(t *testing.T) {
	t.Parallel()
	got := ComputeBotReviewDiagnostics([]any{
		map[string]any{"required": true, "status": BotStatusRateLimited},
	}, "abc123", false)
	assertDiagBool(t, got, "bot_review_completed", false)
	assertDiagBool(t, got, "bot_review_pending", false)
	assertDiagBool(t, got, "bot_review_terminal", false)
	assertDiagBool(t, got, "bot_review_blocked", true)
	assertDiagString(t, got, "bot_review_block_reason", BotStatusRateLimited)
	assertDiagBool(t, got, "bot_review_failed", false)
}

func TestComputeBotReviewDiagnostics_requiredReviewPausedBlocked(t *testing.T) {
	t.Parallel()
	got := ComputeBotReviewDiagnostics([]any{
		map[string]any{"required": true, "status": BotStatusReviewPaused},
	}, "abc123", false)
	assertDiagBool(t, got, "bot_review_completed", false)
	assertDiagBool(t, got, "bot_review_pending", false)
	assertDiagBool(t, got, "bot_review_terminal", false)
	assertDiagBool(t, got, "bot_review_blocked", true)
	assertDiagString(t, got, "bot_review_block_reason", BotStatusReviewPaused)
	assertDiagBool(t, got, "bot_review_failed", false)
}

func TestComputeBotReviewDiagnostics_requiredMixedBlockedReasons(t *testing.T) {
	t.Parallel()
	// Given: two required bots blocked for different reasons (rate-limited + review-paused)
	// When: ComputeBotReviewDiagnostics aggregates them
	// Then: blocked=true and reason collapses to "mixed" so downstream guards see a single
	// non-terminal blocker string; terminal stays false and failed/pending stay false.
	got := ComputeBotReviewDiagnostics([]any{
		map[string]any{"required": true, "status": BotStatusRateLimited},
		map[string]any{"required": true, "status": BotStatusReviewPaused},
	}, "abc123", false)
	assertDiagBool(t, got, "bot_review_completed", false)
	assertDiagBool(t, got, "bot_review_pending", false)
	assertDiagBool(t, got, "bot_review_terminal", false)
	assertDiagBool(t, got, "bot_review_blocked", true)
	assertDiagString(t, got, "bot_review_block_reason", "mixed")
	assertDiagBool(t, got, "bot_review_failed", false)
}

func TestComputeBotReviewDiagnostics_requiredCompletedStale(t *testing.T) {
	t.Parallel()
	// Given: completed bot review on an older commit
	got := ComputeBotReviewDiagnostics([]any{
		map[string]any{"required": true, "status": BotStatusCompleted, "review_commit_sha": "old-sha"},
	}, "new-sha", false)
	// Then: stale is true
	if got["bot_review_stale"].(bool) != true {
		t.Fatalf("mismatched SHA should be stale: %+v", got)
	}
	if got["bot_review_completed"].(bool) != true {
		t.Fatalf("still completed despite staleness: %+v", got)
	}
}

func TestComputeBotReviewDiagnostics_requiredCompletedMissingSHA(t *testing.T) {
	t.Parallel()
	// Given: completed bot review with no review_commit_sha
	got := ComputeBotReviewDiagnostics([]any{
		map[string]any{"required": true, "status": BotStatusCompleted},
	}, "abc123", false)
	// Then: stale is true (fail-closed: missing SHA treated as stale)
	if got["bot_review_stale"].(bool) != true {
		t.Fatalf("missing review SHA should be stale: %+v", got)
	}
}

func TestComputeBotReviewDiagnostics_optionalIgnoredForAggregate(t *testing.T) {
	t.Parallel()
	got := ComputeBotReviewDiagnostics([]any{
		map[string]any{"required": false, "status": BotStatusPending},
	}, "abc123", false)
	if got["bot_review_completed"].(bool) != true {
		t.Fatalf("optional should not block: %+v", got)
	}
	if got["duplicate_findings_detected"].(bool) {
		t.Fatalf("no duplicate signal: %+v", got)
	}
}

func TestComputeBotReviewDiagnostics_duplicateFindingsOptionalOnlyNoRequired(t *testing.T) {
	t.Parallel()
	// Given: no required bots; optional bot reports duplicate findings in review summary.
	// When:  vacuous completion branch runs (sawRequired == false).
	// Then:  duplicate_findings_detected is false (aggregate aligns with required-bot-only semantics).
	got := ComputeBotReviewDiagnostics([]any{
		map[string]any{"required": false, "status": BotStatusCompleted, "duplicate_findings_count": 2},
	}, "abc123", false)
	if got["bot_review_completed"].(bool) != true {
		t.Fatalf("vacuous completion expected: %+v", got)
	}
	if got["duplicate_findings_detected"].(bool) {
		t.Fatalf("want duplicate_findings_detected=false when only optional bots report duplicates, got %+v", got)
	}
}

func TestComputeBotReviewDiagnostics_duplicateFindingsZeroCount(t *testing.T) {
	t.Parallel()
	got := ComputeBotReviewDiagnostics([]any{
		map[string]any{
			"required":                 true,
			"status":                   BotStatusCompleted,
			"review_commit_sha":        "abc123",
			"duplicate_findings_count": 0,
		},
	}, "abc123", false)
	if got["duplicate_findings_detected"].(bool) {
		t.Fatalf("want duplicate_findings_detected=false for zero count, got %+v", got)
	}
}

func TestComputeBotReviewDiagnostics_nonThreadFindingsRequired(t *testing.T) {
	t.Parallel()
	// Given: required bot completed with actionable findings signal
	got := ComputeBotReviewDiagnostics([]any{
		map[string]any{
			"required": true, "status": BotStatusCompleted, "review_commit_sha": "abc123",
			"actionable_findings_count": 1,
		},
	}, "abc123", false)
	// Then: aggregate non_thread_findings_present is true
	if got["non_thread_findings_present"].(bool) != true {
		t.Fatalf("want non_thread_findings_present=true: %+v", got)
	}
}

func TestComputeBotReviewDiagnostics_nonThreadFindingsOptionalOnly(t *testing.T) {
	t.Parallel()
	// Given: optional-only bot with outside-diff findings signal
	got := ComputeBotReviewDiagnostics([]any{
		map[string]any{"required": false, "status": BotStatusCompleted, "outside_diff_findings_count": 3},
	}, "abc123", false)
	// Then: aggregate non_thread_findings_present stays false (optional bots excluded)
	if got["non_thread_findings_present"].(bool) {
		t.Fatalf("optional bot should not set aggregate non-thread flag: %+v", got)
	}
}

func TestComputeBotReviewDiagnostics_nonThreadOutsideDiffRequired(t *testing.T) {
	t.Parallel()
	got := ComputeBotReviewDiagnostics([]any{
		map[string]any{
			"required": true, "status": BotStatusCompleted, "review_commit_sha": "abc123",
			"outside_diff_findings_count": 2,
		},
	}, "abc123", false)
	if got["non_thread_findings_present"].(bool) != true {
		t.Fatalf("want non_thread_findings_present=true: %+v", got)
	}
}

func TestComputeBotReviewDiagnostics_nonThreadDuplicateOnlyDoesNotBlockMergeSignal(t *testing.T) {
	t.Parallel()
	// Given: required bot with duplicate-suppressed count from review summary only (no actionable/outside/issue-comment findings).
	got := ComputeBotReviewDiagnostics([]any{
		map[string]any{
			"required": true, "status": BotStatusCompleted, "review_commit_sha": "abc123",
			"duplicate_findings_count": 1,
		},
	}, "abc123", false)
	if got["non_thread_findings_present"].(bool) {
		t.Fatalf("want non_thread_findings_present=false for duplicate-only: %+v", got)
	}
	if got["duplicate_findings_detected"].(bool) != true {
		t.Fatalf("want duplicate_findings_detected=true for observability: %+v", got)
	}
}

func TestComputeBotReviewDiagnostics_nonThreadDuplicatePlusActionableBlocks(t *testing.T) {
	t.Parallel()
	got := ComputeBotReviewDiagnostics([]any{
		map[string]any{
			"required": true, "status": BotStatusCompleted, "review_commit_sha": "abc123",
			"duplicate_findings_count":  1,
			"actionable_findings_count": 1,
		},
	}, "abc123", false)
	if got["non_thread_findings_present"].(bool) != true {
		t.Fatalf("want non_thread_findings_present=true: %+v", got)
	}
}

func TestComputeBotReviewDiagnostics_nonThreadFindingConversationRequired(t *testing.T) {
	t.Parallel()
	got := ComputeBotReviewDiagnostics([]any{
		map[string]any{
			"required": true, "status": BotStatusCompleted, "review_commit_sha": "abc123",
			"finding_conversation_comments_count": 1,
		},
	}, "abc123", false)
	if got["non_thread_findings_present"].(bool) != true {
		t.Fatalf("want non_thread_findings_present=true: %+v", got)
	}
}

func TestComputeBotReviewDiagnostics_conversationIncompleteFailClosed(t *testing.T) {
	t.Parallel()
	// Given: required bot with zero non-thread counts but conversation comment window incomplete
	got := ComputeBotReviewDiagnostics([]any{
		map[string]any{"required": true, "status": BotStatusCompleted, "review_commit_sha": "abc123"},
	}, "abc123", true)
	if got["non_thread_findings_present"].(bool) != true {
		t.Fatalf("want fail-closed non_thread_findings_present when incomplete: %+v", got)
	}
}

func TestComputeBotReviewDiagnostics_conversationIncompleteOptionalOnly(t *testing.T) {
	t.Parallel()
	// Given: incomplete pagination but only optional bots — aggregate non-thread stays vacuously false
	got := ComputeBotReviewDiagnostics([]any{
		map[string]any{"required": false, "status": BotStatusCompleted},
	}, "abc123", true)
	if got["non_thread_findings_present"].(bool) {
		t.Fatalf("optional-only should not set non_thread even when incomplete: %+v", got)
	}
}

func TestComputeBotReviewDiagnostics_duplicateFindingsOnOptionalBot(t *testing.T) {
	t.Parallel()
	// Given: optional bot reports duplicates; required bot does not.
	// Then: duplicate_findings_detected is false (required-bot-only aggregate).
	got := ComputeBotReviewDiagnostics([]any{
		map[string]any{"required": false, "status": BotStatusCompleted, "duplicate_findings_count": 2},
		map[string]any{"required": true, "status": BotStatusCompleted, "review_commit_sha": "abc123"},
	}, "abc123", false)
	if got["duplicate_findings_detected"].(bool) {
		t.Fatalf("want duplicate_findings_detected=false when only optional has duplicates, got %+v", got)
	}
	if got["bot_review_completed"].(bool) != true {
		t.Fatalf("required bot still completed: %+v", got)
	}
}

func TestComputeBotReviewDiagnostics_duplicateFindingsRequiredBot(t *testing.T) {
	t.Parallel()
	got := ComputeBotReviewDiagnostics([]any{
		map[string]any{
			"required": true, "status": BotStatusCompleted, "review_commit_sha": "abc123",
			"duplicate_findings_count": 1,
		},
	}, "abc123", false)
	if got["duplicate_findings_detected"].(bool) != true {
		t.Fatalf("want duplicate_findings_detected=true for required bot, got %+v", got)
	}
}

func TestClassifyBotStatusGeneric(t *testing.T) {
	t.Parallel()
	if g := classifyBotStatusGeneric(map[string]any{"contains_rate_limit": true}); g != BotStatusRateLimited {
		t.Fatalf("got %q", g)
	}
	if g := classifyBotStatusGeneric(map[string]any{"has_review": true}); g != BotStatusCompleted {
		t.Fatalf("got %q", g)
	}
	if g := classifyBotStatusGeneric(map[string]any{"latest_comment_at": "2026-01-01T00:00:00Z"}); g != BotStatusPending {
		t.Fatalf("got %q", g)
	}
	if g := classifyBotStatusGeneric(map[string]any{}); g != BotStatusNotTriggered {
		t.Fatalf("got %q", g)
	}
}

func TestClassifyBotStatus_usesGenericWithoutClassifierPlugin(t *testing.T) {
	t.Parallel()
	got := ClassifyBotStatus(map[string]any{"contains_review_failed": true}, nil)
	if got != BotStatusReviewFailed {
		t.Fatalf("got %q", got)
	}
}
