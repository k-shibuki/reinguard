package guard

import (
	"strings"
	"testing"

	"github.com/k-shibuki/reinguard/internal/config"
)

func TestEvalWithRules_noDeclarativeRunsBuiltin(t *testing.T) {
	t.Parallel()
	reg := NewRegistry()
	reg.Register(mergeReadinessGuard{})
	s := map[string]any{
		"git": map[string]any{"working_tree_clean": true},
		"github": map[string]any{
			"ci":      map[string]any{"ci_status": "success"},
			"reviews": map[string]any{"review_threads_unresolved": 0},
		},
	}
	r := EvalWithRules(nil, reg, "merge-readiness", s, nil)
	if !r.OK {
		t.Fatalf("%+v", r)
	}
}

func TestEvalWithRules_declarativeNoMatch_skipsBuiltin(t *testing.T) {
	t.Parallel()
	reg := NewRegistry()
	reg.Register(mergeReadinessGuard{})
	rules := []config.Rule{
		{
			Type:     "guard",
			ID:       "g1",
			Priority: 10,
			GuardID:  "merge-readiness",
			When:     map[string]any{"op": "eq", "path": "git.detached_head", "value": false},
		},
	}
	s := map[string]any{
		"git": map[string]any{"working_tree_clean": true, "detached_head": true},
		"github": map[string]any{
			"ci":      map[string]any{"ci_status": "success"},
			"reviews": map[string]any{"review_threads_unresolved": 0},
		},
	}
	r := EvalWithRules(rules, reg, "merge-readiness", s, nil)
	if r.OK || !strings.Contains(r.Reason, "guard rule resolution") {
		t.Fatalf("%+v", r)
	}
}

func TestEvalWithRules_declarativeMatch_runsBuiltin(t *testing.T) {
	t.Parallel()
	reg := NewRegistry()
	reg.Register(mergeReadinessGuard{})
	rules := []config.Rule{
		{
			Type:     "guard",
			ID:       "g1",
			Priority: 10,
			GuardID:  "merge-readiness",
			When:     map[string]any{"op": "eq", "path": "git.detached_head", "value": false},
		},
	}
	s := map[string]any{
		"git": map[string]any{"working_tree_clean": true, "detached_head": false},
		"github": map[string]any{
			"ci":      map[string]any{"ci_status": "success"},
			"reviews": map[string]any{"review_threads_unresolved": 0},
		},
	}
	r := EvalWithRules(rules, reg, "merge-readiness", s, nil)
	if !r.OK {
		t.Fatalf("%+v", r)
	}
}
