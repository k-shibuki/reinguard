package guard

import (
	"fmt"
	"strings"
)

// MergeReadinessResult is JSON output for merge-readiness guard.
type MergeReadinessResult struct {
	GuardID string `json:"guard_id"`
	Reason  string `json:"reason,omitempty"`
	OK      bool   `json:"ok"`
}

// EvalMergeReadiness checks coarse substrate signals (Phase 1).
func EvalMergeReadiness(signals map[string]any) MergeReadinessResult {
	const id = "merge-readiness"
	git := mapString(signals, "git")
	gh := mapString(signals, "github")
	ci := mapString(gh, "ci")
	reviews := mapString(gh, "reviews")

	clean, _ := git["working_tree_clean"].(bool)
	if !clean {
		return MergeReadinessResult{GuardID: id, OK: false, Reason: "git working tree not clean"}
	}
	status, _ := ci["ci_status"].(string)
	status = strings.ToLower(status)
	if status != "success" {
		return MergeReadinessResult{GuardID: id, OK: false, Reason: fmt.Sprintf("ci status is %q, want success", status)}
	}
	unres := 0
	if v, ok := reviews["review_threads_unresolved"]; ok {
		switch x := v.(type) {
		case int:
			unres = x
		case float64:
			unres = int(x)
		}
	}
	if unres != 0 {
		return MergeReadinessResult{GuardID: id, OK: false, Reason: fmt.Sprintf("unresolved review threads: %d", unres)}
	}
	return MergeReadinessResult{GuardID: id, OK: true}
}

func mapString(m map[string]any, k string) map[string]any {
	if m == nil {
		return map[string]any{}
	}
	v, ok := m[k].(map[string]any)
	if !ok {
		return map[string]any{}
	}
	return v
}
