// Package guard evaluates coarse guard predicates over flattened observation signals for the
// semantic control plane (ADR-0011). Built-ins implement [Guard] and register on [DefaultRegistry];
// declarative rules in control/guards/*.yaml gate when a built-in runs ([EvalWithRules]).
// The merge-readiness built-in checks git cleanliness, review blockers first,
// PR mergeability, then CI success. It treats bot blocked states separately from
// bot failures so "CI green" never reads as merge-safe by itself.
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
// github.reviews.review_threads_unresolved must be present, parseable as an integer
// (int, int64, or JSON float64 per signals.GetInt), and zero,
// github.reviews.bot_review_diagnostics.bot_review_pending must be false (fail closed when missing),
// github.reviews.bot_review_diagnostics.bot_review_blocked must be false (fail closed when missing),
// github.reviews.bot_review_diagnostics.bot_review_block_reason must be present when blocked,
// github.reviews.bot_review_diagnostics.non_thread_findings_present must be false (fail closed when missing),
// github.reviews.bot_review_diagnostics.bot_review_terminal must be true (fail closed when missing),
// github.reviews.bot_review_diagnostics.bot_review_failed must be false (fail closed when missing),
// github.reviews.bot_review_diagnostics.bot_review_stale must be false (fail closed when missing),
// github.reviews.review_decisions_changes_requested must be zero (fail closed when missing),
// github.reviews.pagination_incomplete must be false (fail closed when missing), and
// github.reviews.review_decisions_truncated must be false (fail closed when missing),
// github.pull_requests.merge_state_status must be "clean", and github.ci.ci_status must be
// "success" (case-insensitive). Missing or invalid values fail closed.
func EvalMergeReadiness(sigs map[string]any) MergeReadinessResult {
	return evalMergeReadiness(sigs)
}

func evalMergeReadiness(sigs map[string]any) MergeReadinessResult {
	const id = "merge-readiness"

	if reason := checkGit(sigs); reason != "" {
		return MergeReadinessResult{GuardID: id, OK: false, Reason: reason}
	}
	if reason := checkReviewSignals(sigs); reason != "" {
		return MergeReadinessResult{GuardID: id, OK: false, Reason: reason}
	}
	if reason := checkMergeState(sigs); reason != "" {
		return MergeReadinessResult{GuardID: id, OK: false, Reason: reason}
	}
	if reason := checkCI(sigs); reason != "" {
		return MergeReadinessResult{GuardID: id, OK: false, Reason: reason}
	}
	return MergeReadinessResult{GuardID: id, OK: true}
}

func checkGit(sigs map[string]any) string {
	clean, hasClean := signals.GetBool(sigs, "git.working_tree_clean")
	if !hasClean {
		return "missing or invalid git.working_tree_clean"
	}
	if !clean {
		return "git working tree not clean"
	}
	return ""
}

func checkMergeState(sigs map[string]any) string {
	mergeState, hasMergeState := signals.GetString(sigs, "github.pull_requests.merge_state_status")
	if !hasMergeState {
		return "missing or invalid github.pull_requests.merge_state_status"
	}
	if strings.ToLower(mergeState) != "clean" {
		return fmt.Sprintf("merge state status is %q, want clean", mergeState)
	}
	return ""
}

func checkCI(sigs map[string]any) string {
	status, hasStatus := signals.GetString(sigs, "github.ci.ci_status")
	if !hasStatus {
		return "missing or invalid github.ci.ci_status"
	}
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

	if reason := checkBotReviewDiagnostics(sigs); reason != "" {
		return reason
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

func checkBotReviewDiagnostics(sigs map[string]any) string {
	if botPending, ok := signals.GetBool(sigs, "github.reviews.bot_review_diagnostics.bot_review_pending"); !ok {
		return "missing github.reviews.bot_review_diagnostics.bot_review_pending (fail closed)"
	} else if botPending {
		return "required bot review still pending"
	}
	if reason := checkBlockedBotReview(sigs); reason != "" {
		return reason
	}
	if nonThread, ok := signals.GetBool(sigs, "github.reviews.bot_review_diagnostics.non_thread_findings_present"); !ok {
		return "missing github.reviews.bot_review_diagnostics.non_thread_findings_present (fail closed)"
	} else if nonThread {
		return "non-thread review findings present for a required bot"
	}
	if botTerminal, ok := signals.GetBool(sigs, "github.reviews.bot_review_diagnostics.bot_review_terminal"); !ok {
		return "missing github.reviews.bot_review_diagnostics.bot_review_terminal (fail closed)"
	} else if !botTerminal {
		return "required bot review not terminal"
	}
	if botFailed, ok := signals.GetBool(sigs, "github.reviews.bot_review_diagnostics.bot_review_failed"); !ok {
		return "missing github.reviews.bot_review_diagnostics.bot_review_failed (fail closed)"
	} else if botFailed {
		return "required bot review failed"
	}
	if botStale, ok := signals.GetBool(sigs, "github.reviews.bot_review_diagnostics.bot_review_stale"); !ok {
		return "missing github.reviews.bot_review_diagnostics.bot_review_stale (fail closed)"
	} else if botStale {
		return "required bot review is stale or missing review commit SHA"
	}
	return ""
}

func checkBlockedBotReview(sigs map[string]any) string {
	botBlocked, ok := signals.GetBool(sigs, "github.reviews.bot_review_diagnostics.bot_review_blocked")
	if !ok {
		return "missing github.reviews.bot_review_diagnostics.bot_review_blocked (fail closed)"
	}
	if !botBlocked {
		return ""
	}

	reason, hasReason := signals.GetString(sigs, "github.reviews.bot_review_diagnostics.bot_review_block_reason")
	if !hasReason {
		return "missing github.reviews.bot_review_diagnostics.bot_review_block_reason (fail closed)"
	}
	reason = strings.ToLower(strings.TrimSpace(reason))
	if reason == "" {
		return "missing github.reviews.bot_review_diagnostics.bot_review_block_reason (fail closed)"
	}
	switch reason {
	case "rate_limited":
		return "required bot review rate-limited"
	case "review_paused":
		return "required bot review paused"
	case "mixed":
		return "required bot review blocked by multiple reasons"
	default:
		return fmt.Sprintf("required bot review blocked (%s)", reason)
	}
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
