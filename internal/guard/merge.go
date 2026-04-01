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

// EvalMergeReadiness checks merge signals: git working_tree_clean must be true,
// github.ci.ci_status must be "success" (case-insensitive),
// github.reviews.review_threads_unresolved must be present, parseable as an integer
// (int, int64, or JSON float64 per signals.GetInt), and zero,
// github.reviews.bot_review_diagnostics.bot_review_pending must be false (fail closed when missing),
// github.reviews.bot_review_diagnostics.bot_review_terminal must be true (fail closed when missing),
// github.reviews.bot_review_diagnostics.bot_review_failed must be false (fail closed when missing),
// github.reviews.review_decisions_changes_requested must be zero (fail closed when missing),
// github.reviews.pagination_incomplete must be false (fail closed when missing), and
// github.reviews.review_decisions_truncated must be false (fail closed when missing).
// Missing or invalid values fail closed. Missing top-level keys still yield empty nested maps
// for other paths, so absent git / CI fields fail the clean/CI checks as before.
func EvalMergeReadiness(sigs map[string]any) MergeReadinessResult {
	return evalMergeReadiness(sigs)
}

func evalMergeReadiness(sigs map[string]any) MergeReadinessResult {
	const id = "merge-readiness"

	if reason := checkGitAndCI(sigs); reason != "" {
		return MergeReadinessResult{GuardID: id, OK: false, Reason: reason}
	}
	if reason := checkReviewSignals(sigs); reason != "" {
		return MergeReadinessResult{GuardID: id, OK: false, Reason: reason}
	}
	return MergeReadinessResult{GuardID: id, OK: true}
}

func checkGitAndCI(sigs map[string]any) string {
	clean, _ := signals.GetBool(sigs, "git.working_tree_clean")
	if !clean {
		return "git working tree not clean"
	}
	status, _ := signals.GetString(sigs, "github.ci.ci_status")
	if strings.ToLower(status) != "success" {
		return fmt.Sprintf("ci status is %q, want success", status)
	}
	return ""
}

func checkReviewSignals(sigs map[string]any) string {
	unres, ok := signals.GetInt(sigs, "github.reviews.review_threads_unresolved")
	if !ok {
		return "missing or invalid github.reviews.review_threads_unresolved"
	}
	if unres != 0 {
		return fmt.Sprintf("unresolved review threads: %d", unres)
	}

	botPending, hasBotPending := signals.GetBool(sigs, "github.reviews.bot_review_diagnostics.bot_review_pending")
	if !hasBotPending {
		return "missing github.reviews.bot_review_diagnostics.bot_review_pending (fail closed)"
	}
	if botPending {
		return "required bot review still pending"
	}

	botTerminal, hasBotTerminal := signals.GetBool(sigs, "github.reviews.bot_review_diagnostics.bot_review_terminal")
	if !hasBotTerminal {
		return "missing github.reviews.bot_review_diagnostics.bot_review_terminal (fail closed)"
	}
	if !botTerminal {
		return "required bot review not terminal"
	}

	botFailed, hasBotFailed := signals.GetBool(sigs, "github.reviews.bot_review_diagnostics.bot_review_failed")
	if !hasBotFailed {
		return "missing github.reviews.bot_review_diagnostics.bot_review_failed (fail closed)"
	}
	if botFailed {
		return "required bot review failed"
	}

	changesReq, hasChangesReq := signals.GetInt(sigs, "github.reviews.review_decisions_changes_requested")
	if !hasChangesReq {
		return "missing or invalid github.reviews.review_decisions_changes_requested"
	}
	if changesReq != 0 {
		return fmt.Sprintf("changes requested: %d", changesReq)
	}

	if reason := checkReviewDataCompleteness(sigs); reason != "" {
		return reason
	}
	return ""
}

func checkReviewDataCompleteness(sigs map[string]any) string {
	pagIncomplete, hasPagIncomplete := signals.GetBool(sigs, "github.reviews.pagination_incomplete")
	if !hasPagIncomplete {
		return "missing github.reviews.pagination_incomplete (fail closed)"
	}
	if pagIncomplete {
		return "review thread pagination incomplete"
	}

	decTruncated, hasDecTruncated := signals.GetBool(sigs, "github.reviews.review_decisions_truncated")
	if !hasDecTruncated {
		return "missing github.reviews.review_decisions_truncated (fail closed)"
	}
	if decTruncated {
		return "review decisions truncated"
	}
	return ""
}
