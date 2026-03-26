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
	baseSignals := map[string]any{
		"git": map[string]any{"working_tree_clean": true},
		"github": map[string]any{
			"ci":      map[string]any{"ci_status": "success"},
			"reviews": map[string]any{"review_threads_unresolved": 0},
		},
	}
	detachedSignals := map[string]any{
		"git": map[string]any{"working_tree_clean": true, "detached_head": true},
		"github": map[string]any{
			"ci":      map[string]any{"ci_status": "success"},
			"reviews": map[string]any{"review_threads_unresolved": 0},
		},
	}
	attachedSignals := map[string]any{
		"git": map[string]any{"working_tree_clean": true, "detached_head": false},
		"github": map[string]any{
			"ci":      map[string]any{"ci_status": "success"},
			"reviews": map[string]any{"review_threads_unresolved": 0},
		},
	}
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
			// Given: a registry with merge-readiness and optional declarative rules
			reg := NewRegistry()
			if err := reg.Register(mergeReadinessGuard{}); err != nil {
				t.Fatal(err)
			}
			// When: EvalWithRules runs
			got := EvalWithRules(tt.rules, reg, "merge-readiness", tt.signals, nil)
			// Then: outcome matches expectations
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
