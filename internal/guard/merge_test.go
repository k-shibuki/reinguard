package guard

import (
	"strings"
	"testing"
)

func fullReadySignals() map[string]any {
	return map[string]any{
		"git": map[string]any{"working_tree_clean": true},
		"github": map[string]any{
			"ci": map[string]any{"ci_status": "success"},
			"reviews": map[string]any{
				"review_threads_unresolved":          0,
				"review_decisions_changes_requested": 0,
				"pagination_incomplete":              false,
				"review_decisions_truncated":         false,
				"bot_review_diagnostics": map[string]any{
					"bot_review_pending":  false,
					"bot_review_terminal": true,
					"bot_review_failed":   false,
					"bot_review_stale":    false,
				},
			},
		},
	}
}

func TestEvalMergeReadiness_ok(t *testing.T) {
	t.Parallel()
	s := fullReadySignals()
	r := EvalMergeReadiness(s)
	if !r.OK || r.GuardID != "merge-readiness" {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_dirty(t *testing.T) {
	t.Parallel()
	s := map[string]any{
		"git": map[string]any{"working_tree_clean": false},
	}
	r := EvalMergeReadiness(s)
	if r.OK || !strings.Contains(r.Reason, "working tree not clean") {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_missingGitKey(t *testing.T) {
	t.Parallel()
	s := map[string]any{
		"github": map[string]any{
			"ci":      map[string]any{"ci_status": "success"},
			"reviews": map[string]any{"review_threads_unresolved": 0},
		},
	}
	r := EvalMergeReadiness(s)
	if r.OK || !strings.Contains(r.Reason, "working tree not clean") {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_ciFailure(t *testing.T) {
	t.Parallel()
	s := map[string]any{
		"git":    map[string]any{"working_tree_clean": true},
		"github": map[string]any{"ci": map[string]any{"ci_status": "failure"}},
	}
	r := EvalMergeReadiness(s)
	if r.OK || !strings.Contains(r.Reason, `ci status is "failure"`) {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_ciEmptyStatus(t *testing.T) {
	t.Parallel()
	s := map[string]any{
		"git":    map[string]any{"working_tree_clean": true},
		"github": map[string]any{"ci": map[string]any{"ci_status": ""}},
	}
	r := EvalMergeReadiness(s)
	if r.OK || !strings.Contains(r.Reason, `ci status is ""`) {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_unresolvedReviews(t *testing.T) {
	t.Parallel()
	s := map[string]any{
		"git": map[string]any{"working_tree_clean": true},
		"github": map[string]any{
			"ci":      map[string]any{"ci_status": "success"},
			"reviews": map[string]any{"review_threads_unresolved": 3},
		},
	}
	r := EvalMergeReadiness(s)
	if r.OK || !strings.Contains(r.Reason, "unresolved review threads: 3") {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_unresolvedFloat64(t *testing.T) {
	t.Parallel()
	s := map[string]any{
		"git": map[string]any{"working_tree_clean": true},
		"github": map[string]any{
			"ci":      map[string]any{"ci_status": "success"},
			"reviews": map[string]any{"review_threads_unresolved": float64(2)},
		},
	}
	r := EvalMergeReadiness(s)
	if r.OK || !strings.Contains(r.Reason, "unresolved review threads: 2") {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_missingReviewThreadsUnresolved(t *testing.T) {
	t.Parallel()
	s := map[string]any{
		"git": map[string]any{"working_tree_clean": true},
		"github": map[string]any{
			"ci":      map[string]any{"ci_status": "success"},
			"reviews": map[string]any{},
		},
	}
	r := EvalMergeReadiness(s)
	if r.OK || !strings.Contains(r.Reason, "missing or invalid github.reviews.review_threads_unresolved") {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_invalidReviewThreadsUnresolved(t *testing.T) {
	t.Parallel()
	s := map[string]any{
		"git": map[string]any{"working_tree_clean": true},
		"github": map[string]any{
			"ci":      map[string]any{"ci_status": "success"},
			"reviews": map[string]any{"review_threads_unresolved": "two"},
		},
	}
	r := EvalMergeReadiness(s)
	if r.OK || !strings.Contains(r.Reason, "missing or invalid github.reviews.review_threads_unresolved") {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_botReviewPending(t *testing.T) {
	t.Parallel()
	s := fullReadySignals()
	s["github"].(map[string]any)["reviews"].(map[string]any)["bot_review_diagnostics"] = map[string]any{
		"bot_review_pending": true, "bot_review_terminal": false, "bot_review_failed": false, "bot_review_stale": false,
	}
	r := EvalMergeReadiness(s)
	if r.OK || !strings.Contains(r.Reason, "required bot review still pending") {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_botReviewNotTerminal(t *testing.T) {
	t.Parallel()
	s := fullReadySignals()
	s["github"].(map[string]any)["reviews"].(map[string]any)["bot_review_diagnostics"] = map[string]any{
		"bot_review_pending": false, "bot_review_terminal": false, "bot_review_failed": false, "bot_review_stale": false,
	}
	r := EvalMergeReadiness(s)
	if r.OK || !strings.Contains(r.Reason, "required bot review not terminal") {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_botReviewFailed(t *testing.T) {
	t.Parallel()
	s := fullReadySignals()
	s["github"].(map[string]any)["reviews"].(map[string]any)["bot_review_diagnostics"] = map[string]any{
		"bot_review_pending": false, "bot_review_terminal": true, "bot_review_failed": true, "bot_review_stale": false,
	}
	r := EvalMergeReadiness(s)
	if r.OK || !strings.Contains(r.Reason, "required bot review failed") {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_botReviewStale(t *testing.T) {
	t.Parallel()
	s := fullReadySignals()
	s["github"].(map[string]any)["reviews"].(map[string]any)["bot_review_diagnostics"] = map[string]any{
		"bot_review_pending": false, "bot_review_terminal": true, "bot_review_failed": false, "bot_review_stale": true,
	}
	r := EvalMergeReadiness(s)
	if r.OK || !strings.Contains(r.Reason, "required bot review is stale") {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_botReviewStaleMissing(t *testing.T) {
	t.Parallel()
	s := fullReadySignals()
	s["github"].(map[string]any)["reviews"].(map[string]any)["bot_review_diagnostics"] = map[string]any{
		"bot_review_pending": false, "bot_review_terminal": true, "bot_review_failed": false,
	}
	r := EvalMergeReadiness(s)
	if r.OK || !strings.Contains(r.Reason, "missing github.reviews.bot_review_diagnostics.bot_review_stale (fail closed)") {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_botDiagnosticsMissing(t *testing.T) {
	t.Parallel()
	s := fullReadySignals()
	delete(s["github"].(map[string]any)["reviews"].(map[string]any), "bot_review_diagnostics")
	r := EvalMergeReadiness(s)
	if r.OK || !strings.Contains(r.Reason, "fail closed") {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_changesRequested(t *testing.T) {
	t.Parallel()
	s := fullReadySignals()
	s["github"].(map[string]any)["reviews"].(map[string]any)["review_decisions_changes_requested"] = 1
	r := EvalMergeReadiness(s)
	if r.OK || !strings.Contains(r.Reason, "changes requested: 1") {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_paginationIncomplete(t *testing.T) {
	t.Parallel()
	s := fullReadySignals()
	s["github"].(map[string]any)["reviews"].(map[string]any)["pagination_incomplete"] = true
	r := EvalMergeReadiness(s)
	if r.OK || !strings.Contains(r.Reason, "review thread pagination incomplete") {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_decisionsTruncated(t *testing.T) {
	t.Parallel()
	s := fullReadySignals()
	s["github"].(map[string]any)["reviews"].(map[string]any)["review_decisions_truncated"] = true
	r := EvalMergeReadiness(s)
	if r.OK || !strings.Contains(r.Reason, "review decisions truncated") {
		t.Fatalf("%+v", r)
	}
}
