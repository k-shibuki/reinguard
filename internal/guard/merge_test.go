package guard

import (
	"strings"
	"testing"
)

func TestEvalMergeReadiness_ok(t *testing.T) {
	t.Parallel()
	// Given: clean tree, CI success, no unresolved threads
	s := map[string]any{
		"git": map[string]any{"working_tree_clean": true},
		"github": map[string]any{
			"ci":      map[string]any{"ci_status": "success"},
			"reviews": map[string]any{"review_threads_unresolved": 0},
		},
	}
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
