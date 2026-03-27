// Package guard evaluates coarse guard predicates over flattened observation signals for the
// semantic control plane (ADR-0011). Built-ins implement [Guard] and register on [DefaultRegistry];
// declarative rules in control/guards/*.yaml gate when a built-in runs ([EvalWithRules]).
// Phase 1 merge-readiness inspects git cleanliness, CI status, and unresolved review thread counts.
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

// mergeReadinessGuard is the built-in guard with ID "merge-readiness".
type mergeReadinessGuard struct{}

// ID returns the stable guard identifier "merge-readiness".
func (mergeReadinessGuard) ID() string {
	return "merge-readiness"
}

// Eval runs merge-readiness checks on flattened observation signals.
func (mergeReadinessGuard) Eval(sigs map[string]any) MergeReadinessResult {
	return evalMergeReadiness(sigs)
}

// EvalMergeReadiness checks Phase 1 merge signals: git working_tree_clean must be true,
// github.ci.ci_status must be "success" (case-insensitive), and
// github.reviews.review_threads_unresolved must be present, parseable as an integer
// (int, int64, or JSON float64 per signals.GetInt), and zero. Missing or invalid values
// for that path fail closed. Missing top-level keys still yield empty nested maps for other
// paths, so absent git / CI fields fail the clean/CI checks as before.
func EvalMergeReadiness(sigs map[string]any) MergeReadinessResult {
	return evalMergeReadiness(sigs)
}

func evalMergeReadiness(sigs map[string]any) MergeReadinessResult {
	const id = "merge-readiness"

	clean, _ := signals.GetBool(sigs, "git.working_tree_clean")
	if !clean {
		return MergeReadinessResult{GuardID: id, OK: false, Reason: "git working tree not clean"}
	}

	status, _ := signals.GetString(sigs, "github.ci.ci_status")
	if strings.ToLower(status) != "success" {
		return MergeReadinessResult{GuardID: id, OK: false, Reason: fmt.Sprintf("ci status is %q, want success", status)}
	}

	unres, ok := signals.GetInt(sigs, "github.reviews.review_threads_unresolved")
	if !ok {
		return MergeReadinessResult{
			GuardID: id,
			OK:      false,
			Reason:  "missing or invalid github.reviews.review_threads_unresolved",
		}
	}
	if unres != 0 {
		return MergeReadinessResult{GuardID: id, OK: false, Reason: fmt.Sprintf("unresolved review threads: %d", unres)}
	}

	return MergeReadinessResult{GuardID: id, OK: true}
}
