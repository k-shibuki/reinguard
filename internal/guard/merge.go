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

	"github.com/k-shibuki/reinguard/internal/signals"
)

// MergeReadinessResult is JSON output for merge-readiness guard.
type MergeReadinessResult struct {
	GuardID string `json:"guard_id"`
	Reason  string `json:"reason,omitempty"`
	OK      bool   `json:"ok"`
}

// EvalMergeReadiness checks Phase 1 merge signals: git working_tree_clean must be true,
// github.ci.ci_status must be "success" (case-insensitive), and
// github.reviews.review_threads_unresolved must be zero (int or float64 JSON numbers; absent
// or other types are treated as zero). Missing top-level keys yield empty nested maps, so
// absent git.github fields fail the clean/CI checks as expected.
func EvalMergeReadiness(sigs map[string]any) MergeReadinessResult {
	const id = "merge-readiness"

	clean, _ := signals.GetBool(sigs, "git.working_tree_clean")
	if !clean {
		return MergeReadinessResult{GuardID: id, OK: false, Reason: "git working tree not clean"}
	}

	status, _ := signals.GetString(sigs, "github.ci.ci_status")
	if strings.ToLower(status) != "success" {
		return MergeReadinessResult{GuardID: id, OK: false, Reason: fmt.Sprintf("ci status is %q, want success", status)}
	}

	unres, _ := signals.GetInt(sigs, "github.reviews.review_threads_unresolved")
	if unres != 0 {
		return MergeReadinessResult{GuardID: id, OK: false, Reason: fmt.Sprintf("unresolved review threads: %d", unres)}
	}

	return MergeReadinessResult{GuardID: id, OK: true}
}
