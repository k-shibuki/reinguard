package prquery

import (
	"testing"
)

func TestComputeBotReviewDiagnostics_vacuousNoRequired(t *testing.T) {
	t.Parallel()
	got := ComputeBotReviewDiagnostics([]any{})
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
}

func TestComputeBotReviewDiagnostics_requiredPending(t *testing.T) {
	t.Parallel()
	got := ComputeBotReviewDiagnostics([]any{
		map[string]any{"required": true, "status": BotStatusPending},
	})
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
		map[string]any{"required": true, "status": BotStatusCompleted},
	})
	if got["bot_review_completed"].(bool) != true || got["bot_review_pending"].(bool) != false {
		t.Fatalf("%+v", got)
	}
	if got["bot_review_failed"].(bool) != false || got["bot_review_terminal"].(bool) != true {
		t.Fatalf("%+v", got)
	}
}

func TestComputeBotReviewDiagnostics_optionalIgnoredForAggregate(t *testing.T) {
	t.Parallel()
	got := ComputeBotReviewDiagnostics([]any{
		map[string]any{"required": false, "status": BotStatusPending},
	})
	if got["bot_review_completed"].(bool) != true {
		t.Fatalf("optional should not block: %+v", got)
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
