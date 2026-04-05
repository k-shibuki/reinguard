package guard

import (
	"strings"
	"testing"

	"github.com/k-shibuki/reinguard/internal/config"
)

func TestEvalWithRules(t *testing.T) {
	t.Parallel()
	decl := []config.Rule{
		{
			Type:     "guard",
			ID:       "g1",
			Priority: 10,
			GuardID:  "merge-readiness",
			When:     map[string]any{"op": "eq", "path": "git.detached_head", "value": false},
		},
	}
	readyReviews := func() map[string]any {
		return map[string]any{
			"review_threads_unresolved":          0,
			"review_decisions_changes_requested": 0,
			"pagination_incomplete":              false,
			"review_decisions_truncated":         false,
			"bot_review_diagnostics": map[string]any{
				"bot_review_pending":          false,
				"bot_review_terminal":         true,
				"bot_review_failed":           false,
				"bot_review_stale":            false,
				"non_thread_findings_present": false,
			},
		}
	}
	baseSignals := map[string]any{
		"git": map[string]any{"working_tree_clean": true},
		"github": map[string]any{
			"ci":      map[string]any{"ci_status": "success"},
			"reviews": readyReviews(),
		},
	}
	detachedSignals := map[string]any{
		"git": map[string]any{"working_tree_clean": true, "detached_head": true},
		"github": map[string]any{
			"ci":      map[string]any{"ci_status": "success"},
			"reviews": readyReviews(),
		},
	}
	attachedSignals := map[string]any{
		"git": map[string]any{"working_tree_clean": true, "detached_head": false},
		"github": map[string]any{
			"ci":      map[string]any{"ci_status": "success"},
			"reviews": readyReviews(),
		},
	}
	// Given: merge-readiness guard, optional declarative rules, signal variants
	// When: EvalWithRules runs per row
	// Then: OK and Reason match expectations
	tests := []struct {
		signals          map[string]any
		name             string
		wantReasonSubstr string
		rules            []config.Rule
		wantOK           bool
	}{
		{
			signals: baseSignals,
			name:    "no_declarative_runs_builtin",
			rules:   nil,
			wantOK:  true,
		},
		{
			signals:          detachedSignals,
			name:             "declarative_no_match_skips_builtin",
			wantReasonSubstr: "guard rule resolution",
			rules:            decl,
			wantOK:           false,
		},
		{
			signals: attachedSignals,
			name:    "declarative_match_runs_builtin",
			rules:   decl,
			wantOK:  true,
		},
	}
	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			reg := NewRegistry()
			if err := reg.Register(mergeReadinessGuard{}); err != nil {
				t.Fatal(err)
			}
			got := EvalWithRules(tt.rules, reg, "merge-readiness", tt.signals, nil)
			if tt.wantOK {
				if !got.OK {
					t.Fatalf("want ok, got %+v", got)
				}
				return
			}
			if got.OK {
				t.Fatalf("want failure, got %+v", got)
			}
			if tt.wantReasonSubstr != "" && !strings.Contains(got.Reason, tt.wantReasonSubstr) {
				t.Fatalf("reason %q should contain %q", got.Reason, tt.wantReasonSubstr)
			}
		})
	}
}
