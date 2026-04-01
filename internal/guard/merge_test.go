package guard

import (
	"strings"
	"testing"
)

// fullReadySignals returns a signals map where all merge-readiness checks pass.
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
				},
			},
		},
	}
}

func TestEvalMergeReadiness_ok(t *testing.T) {
	t.Parallel()
	// Given: all signals satisfy merge readiness
	s := fullReadySignals()
	// When: EvalMergeReadiness runs
	r := EvalMergeReadiness(s)
	// Then: OK
	if !r.OK || r.GuardID != "merge-readiness" {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_dirty(t *testing.T) {
	t.Parallel()
	// Given: working tree not clean
	s := map[string]any{
		"git": map[string]any{"working_tree_clean": false},
	}
	// When: EvalMergeReadiness runs
	r := EvalMergeReadiness(s)
	// Then: not OK; reason mentions dirty tree
	if r.OK || !strings.Contains(r.Reason, "working tree not clean") {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_missingGitKey(t *testing.T) {
	t.Parallel()
	// Given: signals with github subtree but no git key (clean defaults false)
	s := map[string]any{
		"github": map[string]any{
			"ci":      map[string]any{"ci_status": "success"},
			"reviews": map[string]any{"review_threads_unresolved": 0},
		},
	}
	// When: EvalMergeReadiness runs
	r := EvalMergeReadiness(s)
	// Then: not OK; working tree treated as not clean
	if r.OK || !strings.Contains(r.Reason, "working tree not clean") {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_ciFailure(t *testing.T) {
	t.Parallel()
	// Given: clean tree but CI failure
	s := map[string]any{
		"git":    map[string]any{"working_tree_clean": true},
		"github": map[string]any{"ci": map[string]any{"ci_status": "failure"}},
	}
	// When: EvalMergeReadiness runs
	r := EvalMergeReadiness(s)
	// Then: not OK; reason cites CI
	if r.OK || !strings.Contains(r.Reason, `ci status is "failure"`) || !strings.Contains(r.Reason, "want success") {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_ciEmptyStatus(t *testing.T) {
	t.Parallel()
	// Given: empty ci_status string
	s := map[string]any{
		"git":    map[string]any{"working_tree_clean": true},
		"github": map[string]any{"ci": map[string]any{"ci_status": ""}},
	}
	// When: EvalMergeReadiness runs
	r := EvalMergeReadiness(s)
	// Then: not OK
	if r.OK || !strings.Contains(r.Reason, `ci status is ""`) {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_unresolvedReviews(t *testing.T) {
	t.Parallel()
	// Given: unresolved review thread count > 0
	s := map[string]any{
		"git": map[string]any{"working_tree_clean": true},
		"github": map[string]any{
			"ci":      map[string]any{"ci_status": "success"},
			"reviews": map[string]any{"review_threads_unresolved": 3},
		},
	}
	// When: EvalMergeReadiness runs
	r := EvalMergeReadiness(s)
	// Then: not OK with count in reason
	if r.OK || !strings.Contains(r.Reason, "unresolved review threads: 3") {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_unresolvedFloat64(t *testing.T) {
	t.Parallel()
	// Given: JSON-decoded number as float64
	s := map[string]any{
		"git": map[string]any{"working_tree_clean": true},
		"github": map[string]any{
			"ci":      map[string]any{"ci_status": "success"},
			"reviews": map[string]any{"review_threads_unresolved": float64(2)},
		},
	}
	// When: EvalMergeReadiness runs
	r := EvalMergeReadiness(s)
	// Then: count coerced from float64 appears in reason
	if r.OK || !strings.Contains(r.Reason, "unresolved review threads: 2") {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_missingReviewThreadsUnresolved(t *testing.T) {
	t.Parallel()
	// Given: clean tree, CI success, but reviews subtree omits review_threads_unresolved
	s := map[string]any{
		"git": map[string]any{"working_tree_clean": true},
		"github": map[string]any{
			"ci":      map[string]any{"ci_status": "success"},
			"reviews": map[string]any{},
		},
	}
	// When: EvalMergeReadiness runs
	r := EvalMergeReadiness(s)
	// Then: not OK; missing signal is fail-closed
	if r.OK || !strings.Contains(r.Reason, "missing or invalid github.reviews.review_threads_unresolved") {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_invalidReviewThreadsUnresolved(t *testing.T) {
	t.Parallel()
	// Given: review_threads_unresolved present but not numeric
	s := map[string]any{
		"git": map[string]any{"working_tree_clean": true},
		"github": map[string]any{
			"ci":      map[string]any{"ci_status": "success"},
			"reviews": map[string]any{"review_threads_unresolved": "two"},
		},
	}
	// When: EvalMergeReadiness runs
	r := EvalMergeReadiness(s)
	// Then: not OK; invalid type is fail-closed
	if r.OK || !strings.Contains(r.Reason, "missing or invalid github.reviews.review_threads_unresolved") {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_botReviewPending(t *testing.T) {
	t.Parallel()
	// Given: bot review is still pending
	s := fullReadySignals()
	s["github"].(map[string]any)["reviews"].(map[string]any)["bot_review_diagnostics"] = map[string]any{
		"bot_review_pending":  true,
		"bot_review_terminal": false,
		"bot_review_failed":   false,
	}
	// When: EvalMergeReadiness runs
	r := EvalMergeReadiness(s)
	// Then: not OK
	if r.OK || !strings.Contains(r.Reason, "required bot review still pending") {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_botReviewNotTerminal(t *testing.T) {
	t.Parallel()
	// Given: bot review is not pending but not terminal (rate-limited/paused)
	s := fullReadySignals()
	s["github"].(map[string]any)["reviews"].(map[string]any)["bot_review_diagnostics"] = map[string]any{
		"bot_review_pending":  false,
		"bot_review_terminal": false,
		"bot_review_failed":   false,
	}
	// When: EvalMergeReadiness runs
	r := EvalMergeReadiness(s)
	// Then: not OK
	if r.OK || !strings.Contains(r.Reason, "required bot review not terminal") {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_botReviewTerminalMissing(t *testing.T) {
	t.Parallel()
	// Given: bot_review_terminal is absent
	s := fullReadySignals()
	s["github"].(map[string]any)["reviews"].(map[string]any)["bot_review_diagnostics"] = map[string]any{
		"bot_review_pending": false,
		"bot_review_failed":  false,
	}
	// When: EvalMergeReadiness runs
	r := EvalMergeReadiness(s)
	// Then: not OK; fail closed
	if r.OK || !strings.Contains(r.Reason, "missing github.reviews.bot_review_diagnostics.bot_review_terminal (fail closed)") {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_botReviewFailed(t *testing.T) {
	t.Parallel()
	// Given: bot review failed
	s := fullReadySignals()
	s["github"].(map[string]any)["reviews"].(map[string]any)["bot_review_diagnostics"] = map[string]any{
		"bot_review_pending":  false,
		"bot_review_terminal": true,
		"bot_review_failed":   true,
	}
	// When: EvalMergeReadiness runs
	r := EvalMergeReadiness(s)
	// Then: not OK
	if r.OK || !strings.Contains(r.Reason, "required bot review failed") {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_botReviewFailedMissing(t *testing.T) {
	t.Parallel()
	// Given: bot_review_failed is absent
	s := fullReadySignals()
	s["github"].(map[string]any)["reviews"].(map[string]any)["bot_review_diagnostics"] = map[string]any{
		"bot_review_pending":  false,
		"bot_review_terminal": true,
	}
	// When: EvalMergeReadiness runs
	r := EvalMergeReadiness(s)
	// Then: not OK; fail closed
	if r.OK || !strings.Contains(r.Reason, "missing github.reviews.bot_review_diagnostics.bot_review_failed (fail closed)") {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_botReviewDiagnosticsMissing(t *testing.T) {
	t.Parallel()
	// Given: bot_review_diagnostics subtree is absent
	s := fullReadySignals()
	delete(s["github"].(map[string]any)["reviews"].(map[string]any), "bot_review_diagnostics")
	// When: EvalMergeReadiness runs
	r := EvalMergeReadiness(s)
	// Then: not OK; fail closed
	if r.OK || !strings.Contains(r.Reason, "fail closed") {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_changesRequested(t *testing.T) {
	t.Parallel()
	// Given: formal changes requested
	s := fullReadySignals()
	s["github"].(map[string]any)["reviews"].(map[string]any)["review_decisions_changes_requested"] = 1
	// When: EvalMergeReadiness runs
	r := EvalMergeReadiness(s)
	// Then: not OK
	if r.OK || !strings.Contains(r.Reason, "changes requested: 1") {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_changesRequestedMissing(t *testing.T) {
	t.Parallel()
	// Given: review_decisions_changes_requested is absent
	s := fullReadySignals()
	delete(s["github"].(map[string]any)["reviews"].(map[string]any), "review_decisions_changes_requested")
	// When: EvalMergeReadiness runs
	r := EvalMergeReadiness(s)
	// Then: not OK; fail closed
	if r.OK || !strings.Contains(r.Reason, "missing or invalid github.reviews.review_decisions_changes_requested") {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_paginationIncomplete(t *testing.T) {
	t.Parallel()
	// Given: pagination was incomplete
	s := fullReadySignals()
	s["github"].(map[string]any)["reviews"].(map[string]any)["pagination_incomplete"] = true
	// When: EvalMergeReadiness runs
	r := EvalMergeReadiness(s)
	// Then: not OK
	if r.OK || !strings.Contains(r.Reason, "review thread pagination incomplete") {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_paginationIncompleteMissing(t *testing.T) {
	t.Parallel()
	// Given: pagination_incomplete key absent
	s := fullReadySignals()
	delete(s["github"].(map[string]any)["reviews"].(map[string]any), "pagination_incomplete")
	// When: EvalMergeReadiness runs
	r := EvalMergeReadiness(s)
	// Then: not OK; fail closed
	if r.OK || !strings.Contains(r.Reason, "fail closed") {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_decisionsTriaged(t *testing.T) {
	t.Parallel()
	// Given: review decisions were truncated
	s := fullReadySignals()
	s["github"].(map[string]any)["reviews"].(map[string]any)["review_decisions_truncated"] = true
	// When: EvalMergeReadiness runs
	r := EvalMergeReadiness(s)
	// Then: not OK
	if r.OK || !strings.Contains(r.Reason, "review decisions truncated") {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_decisionsTruncatedMissing(t *testing.T) {
	t.Parallel()
	// Given: review_decisions_truncated key absent
	s := fullReadySignals()
	delete(s["github"].(map[string]any)["reviews"].(map[string]any), "review_decisions_truncated")
	// When: EvalMergeReadiness runs
	r := EvalMergeReadiness(s)
	// Then: not OK; fail closed
	if r.OK || !strings.Contains(r.Reason, "fail closed") {
		t.Fatalf("%+v", r)
	}
}
