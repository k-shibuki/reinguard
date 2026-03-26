package guard

import (
	"fmt"

	"github.com/k-shibuki/reinguard/internal/config"
	"github.com/k-shibuki/reinguard/internal/resolve"
)

// EvalWithRules evaluates guardID: when declarative rules exist for that guard_id, ResolveGuard
// must succeed before the built-in runs; otherwise the built-in runs alone (backward compatible).
func EvalWithRules(rules []config.Rule, reg *Registry, guardID string, signals map[string]any, degraded map[string]struct{}) MergeReadinessResult {
	if _, ok := reg.Lookup(guardID); !ok {
		return MergeReadinessResult{GuardID: guardID, OK: false, Reason: fmt.Sprintf("unknown guard %q", guardID)}
	}
	if hasGuardRulesForID(rules, guardID) {
		res, _ := resolve.ResolveGuard(rules, signals, degraded, guardID)
		if res.Kind != resolve.OutcomeResolved {
			reason := res.Reason
			if reason == "" {
				reason = fmt.Sprintf("outcome %s", res.Kind)
			} else {
				reason = fmt.Sprintf("%s: %s", res.Kind, reason)
			}
			return MergeReadinessResult{GuardID: guardID, OK: false, Reason: "guard rule resolution " + reason}
		}
	}
	g, _ := reg.Lookup(guardID)
	return g.Eval(signals)
}

func hasGuardRulesForID(rules []config.Rule, guardID string) bool {
	for _, r := range rules {
		if r.Type == "guard" && r.GuardID == guardID {
			return true
		}
	}
	return false
}
