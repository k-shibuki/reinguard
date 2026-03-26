// Package guard evaluates coarse guard predicates over flattened observation signals for the
// semantic control plane (ADR-0011). Phase 1 merge-readiness inspects git cleanliness,
// CI status, and unresolved review thread counts embedded in the signal map.
//
// # Inputs and outputs
//
// Callers pass the same style of flattened map used by context build (keys such as
// git.*, github.ci.*, github.reviews.*). EvalMergeReadiness returns a JSON-friendly
// struct with OK and Reason.
//
// # Error semantics
//
// EvalMergeReadiness does not return errors; failures are expressed as OK == false with
// a human-readable Reason.
//
// ADR-0011 (semantic control plane structure).
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

// EvalMergeReadiness checks coarse substrate signals (Phase 1): working tree clean,
// github.ci.ci_status == "success", and github.reviews.review_threads_unresolved == 0.
// Missing or wrongly-typed nested maps are treated as failing conditions (empty CI status,
// unresolved count defaulting from zero only when absent or numeric).
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
