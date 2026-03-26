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
	r := EvalMergeReadiness(s)
	if r.OK || !strings.Contains(r.Reason, "working tree not clean") {
		t.Fatalf("%+v", r)
	}
}

func TestEvalMergeReadiness_missingGitKey(t *testing.T) {
	t.Parallel()
	// Given: no git key (clean defaults false)
	r := EvalMergeReadiness(map[string]any{
		"github": map[string]any{
			"ci":      map[string]any{"ci_status": "success"},
			"reviews": map[string]any{"review_threads_unresolved": 0},
		},
	})
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
	if r.OK || !strings.Contains(r.Reason, `ci status is "failure"`) || !strings.Contains(r.Reason, "want success") {
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
	// Given: JSON-decoded number as float64
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
