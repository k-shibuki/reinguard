package prquery

import (
	"testing"
)

func TestComputeBotReviewDiagnostics_vacuousNoRequired(t *testing.T) {
	t.Parallel()
	got := ComputeBotReviewDiagnostics([]any{}, "abc123")
	if g, ok := got["bot_review_completed"].(bool); !ok || !g {
		t.Fatalf("completed: %+v", got)
	}
	if g, ok := got["bot_review_pending"].(bool); !ok || g {
		t.Fatalf("pending: %+v", got)
	}
	if g, ok := got["bot_review_terminal"].(bool); !ok || !g {
		t.Fatalf("terminal: %+v", got)
	}
	if g, ok := got["bot_review_failed"].(bool); !ok || g {
		t.Fatalf("failed: %+v", got)
	}
	if g, ok := got["bot_review_stale"].(bool); !ok || g {
		t.Fatalf("stale should be false for vacuous: %+v", got)
	}
	if g, ok := got["duplicate_findings_detected"].(bool); !ok || g {
		t.Fatalf("duplicate_findings_detected should be false for vacuous: %+v", got)
	}
}

func TestComputeBotReviewDiagnostics_requiredPending(t *testing.T) {
	t.Parallel()
	got := ComputeBotReviewDiagnostics([]any{
		map[string]any{"required": true, "status": BotStatusPending},
	}, "abc123")
	if got["bot_review_completed"].(bool) != false {
		t.Fatalf("%+v", got)
	}
	if got["bot_review_pending"].(bool) != true {
		t.Fatalf("%+v", got)
	}
	if got["bot_review_terminal"].(bool) != false {
		t.Fatalf("%+v", got)
	}
	if got["bot_review_failed"].(bool) != false {
		t.Fatalf("%+v", got)
	}
}

func TestComputeBotReviewDiagnostics_requiredCompleted(t *testing.T) {
	t.Parallel()
	got := ComputeBotReviewDiagnostics([]any{
		map[string]any{"required": true, "status": BotStatusCompleted, "review_commit_sha": "abc123"},
	}, "abc123")
	if got["bot_review_completed"].(bool) != true || got["bot_review_pending"].(bool) != false {
		t.Fatalf("%+v", got)
	}
	if got["bot_review_failed"].(bool) != false || got["bot_review_terminal"].(bool) != true {
		t.Fatalf("%+v", got)
	}
	if got["bot_review_stale"].(bool) != false {
		t.Fatalf("matching SHA should not be stale: %+v", got)
	}
}

func TestComputeBotReviewDiagnostics_requiredCompletedStale(t *testing.T) {
	t.Parallel()
	// Given: completed bot review on an older commit
	got := ComputeBotReviewDiagnostics([]any{
		map[string]any{"required": true, "status": BotStatusCompleted, "review_commit_sha": "old-sha"},
	}, "new-sha")
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
	}, "abc123")
	// Then: stale is true (fail-closed: missing SHA treated as stale)
	if got["bot_review_stale"].(bool) != true {
		t.Fatalf("missing review SHA should be stale: %+v", got)
	}
}

func TestComputeBotReviewDiagnostics_optionalIgnoredForAggregate(t *testing.T) {
	t.Parallel()
	got := ComputeBotReviewDiagnostics([]any{
		map[string]any{"required": false, "status": BotStatusPending},
	}, "abc123")
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
	// Then:  duplicate_findings_detected still surfaces for triage.
	got := ComputeBotReviewDiagnostics([]any{
		map[string]any{"required": false, "status": BotStatusCompleted, "cr_duplicate_findings_count": 2},
	}, "abc123")
	if got["bot_review_completed"].(bool) != true {
		t.Fatalf("vacuous completion expected: %+v", got)
	}
	if got["duplicate_findings_detected"].(bool) != true {
		t.Fatalf("want duplicate_findings_detected=true in no-required branch, got %+v", got)
	}
}

func TestComputeBotReviewDiagnostics_duplicateFindingsZeroCount(t *testing.T) {
	t.Parallel()
	got := ComputeBotReviewDiagnostics([]any{
		map[string]any{
			"required":                    true,
			"status":                      BotStatusCompleted,
			"review_commit_sha":           "abc123",
			"cr_duplicate_findings_count": 0,
		},
	}, "abc123")
	if got["duplicate_findings_detected"].(bool) {
		t.Fatalf("want duplicate_findings_detected=false for zero count, got %+v", got)
	}
}

func TestComputeBotReviewDiagnostics_duplicateFindingsOnOptionalBot(t *testing.T) {
	t.Parallel()
	// Given: optional bot with duplicate findings in review summary (observation still surfaces it).
	got := ComputeBotReviewDiagnostics([]any{
		map[string]any{"required": false, "status": BotStatusCompleted, "cr_duplicate_findings_count": 2},
		map[string]any{"required": true, "status": BotStatusCompleted, "review_commit_sha": "abc123"},
	}, "abc123")
	if got["duplicate_findings_detected"].(bool) != true {
		t.Fatalf("want duplicate_findings_detected=true, got %+v", got)
	}
	if got["bot_review_completed"].(bool) != true {
		t.Fatalf("required bot still completed: %+v", got)
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
