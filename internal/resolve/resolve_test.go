package resolve

import (
	"reflect"
	"strings"
	"testing"

	"github.com/k-shibuki/reinguard/internal/config"
)

func TestResolve_unsupportedRuleType(t *testing.T) {
	t.Parallel()
	// Given: an unsupported rule type
	// When: Resolve is called
	// Then: OutcomeUnsupported with re-entry metadata (ADR-0007)
	res, err := Resolve(nil, nil, nil, "not-a-rule-type")
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != OutcomeUnsupported || !strings.Contains(res.Reason, `unsupported rule type "not-a-rule-type"`) {
		t.Fatalf("got %+v", res)
	}
	if len(res.MissingEvidence) != 1 || res.MissingEvidence[0] != "rule_type:not-a-rule-type" {
		t.Fatalf("missing_evidence: %v", res.MissingEvidence)
	}
	if res.ReEntryHint == "" {
		t.Fatal("want ReEntryHint")
	}
}

func TestResolve_stateMatchesResolveState(t *testing.T) {
	t.Parallel()
	// Given: two matching state rules with different priorities
	// When: ResolveState and Resolve(..., "state") run
	// Then: both return the same Result
	rules := []config.Rule{
		{Type: "state", ID: "a", Priority: 20, StateID: "A", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
		{Type: "state", ID: "b", Priority: 10, StateID: "B", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
	}
	signals := map[string]any{"x": 1}
	a, err := ResolveState(rules, signals, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Resolve(rules, signals, nil, "state")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("Resolve vs ResolveState: %+v vs %+v", a, b)
	}
}

func TestResolve_guardMatchesResolveGuard(t *testing.T) {
	t.Parallel()
	// Given: two guard rules for the same guard_id and matching signals
	rules := []config.Rule{
		{Type: "guard", ID: "a", Priority: 20, GuardID: "g1", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
		{Type: "guard", ID: "b", Priority: 10, GuardID: "g1", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
	}
	signals := map[string]any{"x": 1}
	// When: ResolveGuard vs Resolve on pre-scoped rules
	a, err := ResolveGuard(rules, signals, nil, "g1")
	if err != nil {
		t.Fatal(err)
	}
	b, err := Resolve(rules, signals, nil, "guard")
	if err != nil {
		t.Fatal(err)
	}
	// Then: identical results
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("Resolve vs ResolveGuard: %+v vs %+v", a, b)
	}
}

func TestResolveGuard_priorityWins(t *testing.T) {
	t.Parallel()
	// Given: two matching guard rules at different priorities for g1
	rules := []config.Rule{
		{Type: "guard", ID: "a", Priority: 20, GuardID: "g1", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
		{Type: "guard", ID: "b", Priority: 10, GuardID: "g1", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
	}
	// When: ResolveGuard selects for g1
	res, err := ResolveGuard(rules, map[string]any{"x": 1}, nil, "g1")
	if err != nil {
		t.Fatal(err)
	}
	// Then: lower priority number wins (rule b)
	if res.Kind != OutcomeResolved || res.GuardID != "g1" || res.RuleID != "b" {
		t.Fatalf("%+v", res)
	}
}

func TestResolveGuard_otherGuardIDIgnored(t *testing.T) {
	t.Parallel()
	// Given: only rules for guard_id "other" while resolving "g1"
	rules := []config.Rule{
		{Type: "guard", ID: "x", Priority: 10, GuardID: "other", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
	}
	// When: ResolveGuard runs for g1
	res, err := ResolveGuard(rules, map[string]any{"x": 1}, nil, "g1")
	if err != nil {
		t.Fatal(err)
	}
	// Then: degraded — no rule targets g1
	if res.Kind != OutcomeDegraded || !strings.Contains(res.Reason, "no matching guard rule") {
		t.Fatalf("%+v", res)
	}
}

func TestResolve_guardMissingGuardID(t *testing.T) {
	t.Parallel()
	// Given: a winning guard rule with empty guard_id (invalid shape)
	rules := []config.Rule{
		{Type: "guard", ID: "bad", Priority: 10, GuardID: "", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
	}
	// When: Resolve(..., "guard") runs
	res, err := Resolve(rules, map[string]any{"x": 1}, nil, "guard")
	if err != nil {
		t.Fatal(err)
	}
	// Then: unsupported with missing guard_id reason
	if res.Kind != OutcomeUnsupported || !strings.Contains(res.Reason, `guard rule "bad" is missing guard_id`) {
		t.Fatalf("%+v", res)
	}
}

func TestResolve_routeMatchesResolveRoute(t *testing.T) {
	t.Parallel()
	// Given: two matching route rules with different priorities
	// When: ResolveRoute and Resolve(..., "route") run
	// Then: both return the same Result
	rules := []config.Rule{
		{Type: "route", ID: "low", Priority: 5, RouteID: "R5", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
		{Type: "route", ID: "high", Priority: 20, RouteID: "R20", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
	}
	signals := map[string]any{"x": 1}
	a, err := ResolveRoute(rules, signals, nil)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Resolve(rules, signals, nil, "route")
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(a, b) {
		t.Fatalf("Resolve vs ResolveRoute: %+v vs %+v", a, b)
	}
}

func TestResolveState_priorityWins(t *testing.T) {
	t.Parallel()
	// Given: two matching state rules with different priorities
	// When: ResolveState runs
	// Then: lower priority number wins
	rules := []config.Rule{
		{Type: "state", ID: "a", Priority: 20, StateID: "A", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
		{Type: "state", ID: "b", Priority: 10, StateID: "B", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
	}
	signals := map[string]any{"x": 1}
	res, err := ResolveState(rules, signals, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != OutcomeResolved || res.StateID != "B" {
		t.Fatalf("%+v", res)
	}
}

func TestResolveState_excludesHigherPriorityWhenExtraPredicatesFail(t *testing.T) {
	t.Parallel()
	// Given: a higher-priority (lower number) state rule with the same base signals
	// as a bot-wait rule, plus extra "mutual exclusion" predicates on diagnostics (Issue #129)
	// When: diagnostics indicate bot-wait, so the extra predicates on the human rule fail
	// Then: the next matching rule (bot-wait) wins
	notPendingOr := map[string]any{
		"or": []any{
			map[string]any{"op": "not_exists", "path": "github.reviews.bot_review_diagnostics.bot_review_pending"},
			map[string]any{"op": "eq", "path": "github.reviews.bot_review_diagnostics.bot_review_pending", "value": false},
		},
	}
	humanWhen := map[string]any{
		"and": []any{
			map[string]any{"op": "eq", "path": "github.pull_requests.pr_exists_for_branch", "value": true},
			map[string]any{"op": "gt", "path": "github.reviews.review_threads_unresolved", "value": 0},
			notPendingOr,
		},
	}
	rules := []config.Rule{
		{Type: "state", ID: "human", Priority: 8, StateID: "unresolved_threads", When: humanWhen},
		{Type: "state", ID: "bot_run", Priority: 15, StateID: "waiting_bot_run", When: map[string]any{
			"and": []any{
				map[string]any{"op": "eq", "path": "github.pull_requests.pr_exists_for_branch", "value": true},
				map[string]any{"op": "eq", "path": "github.reviews.bot_review_diagnostics.bot_review_pending", "value": true},
			},
		}},
	}
	signals := map[string]any{
		"github": map[string]any{
			"pull_requests": map[string]any{
				"pr_exists_for_branch": true,
			},
			"reviews": map[string]any{
				"review_threads_unresolved": 1,
				"bot_review_diagnostics": map[string]any{
					"bot_review_pending": true,
				},
			},
		},
	}
	res, err := ResolveState(rules, signals, nil)
	if err != nil {
		t.Fatal(err)
	}
	if res.Kind != OutcomeResolved || res.StateID != "waiting_bot_run" {
		t.Fatalf("got %+v", res)
	}
}

func TestResolveState_tieAmbiguous(t *testing.T) {
	t.Parallel()
	// Given: two matches at same best priority
	rules := []config.Rule{
		{Type: "state", ID: "a", Priority: 10, StateID: "A", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
		{Type: "state", ID: "b", Priority: 10, StateID: "B", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
	}
	signals := map[string]any{"x": 1}
	// When: ResolveState runs
	res, err := ResolveState(rules, signals, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Then: ambiguous outcome with reason
	if res.Kind != OutcomeAmbiguous {
		t.Fatalf("%+v", res)
	}
	if !strings.Contains(res.Reason, "same best priority") {
		t.Fatal(res.Reason)
	}
}

func TestResolveState_noMatchDegraded(t *testing.T) {
	t.Parallel()
	// Given: state rules that do not match signals
	// When: ResolveState runs
	rules := []config.Rule{
		{Type: "state", ID: "a", Priority: 10, StateID: "A", When: map[string]any{"op": "eq", "path": "x", "value": 2}},
	}
	res, err := ResolveState(rules, map[string]any{"x": 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Then: degraded with no-match reason
	if res.Kind != OutcomeDegraded || !strings.Contains(res.Reason, "no matching state rule") {
		t.Fatalf("%+v", res)
	}
}

func TestResolveState_suppressed(t *testing.T) {
	t.Parallel()
	// Given: matching rule with depends_on a degraded provider
	rules := []config.Rule{
		{Type: "state", ID: "a", Priority: 10, StateID: "A", When: map[string]any{"op": "eq", "path": "x", "value": 1}, DependsOn: []string{"github"}},
	}
	signals := map[string]any{"x": 1}
	deg := map[string]struct{}{"github": {}}
	// When: ResolveState runs
	res, err := ResolveState(rules, signals, deg)
	if err != nil {
		t.Fatal(err)
	}
	// Then: degraded due to dependency
	if res.Kind != OutcomeDegraded || !strings.Contains(res.Reason, "depends_on") {
		t.Fatalf("%+v", res)
	}
}

func TestResolveState_missingStateID(t *testing.T) {
	t.Parallel()
	// Given: state rule with empty state_id
	rules := []config.Rule{
		{Type: "state", ID: "bad", Priority: 10, StateID: "", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
	}
	// When: ResolveState runs
	res, err := ResolveState(rules, map[string]any{"x": 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Then: unsupported outcome (invalid rule shape)
	if res.Kind != OutcomeUnsupported || !strings.Contains(res.Reason, `state rule "bad" is missing state_id`) {
		t.Fatalf("%+v", res)
	}
	if len(res.MissingEvidence) < 2 || res.MissingEvidence[0] != "rule_id:bad" {
		t.Fatalf("missing_evidence: %v", res.MissingEvidence)
	}
}

func TestResolveState_ruleEvalError(t *testing.T) {
	t.Parallel()
	// Given: state rule with invalid when (eval error)
	rules := []config.Rule{
		{Type: "state", ID: "r", Priority: 10, StateID: "S", When: map[string]any{"op": "bogus"}},
	}
	// When: ResolveState runs
	res, err := ResolveState(rules, map[string]any{}, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Then: unsupported — substrate cannot interpret the when-clause safely
	if res.Kind != OutcomeUnsupported || !strings.Contains(res.Reason, "rule r:") {
		t.Fatalf("%+v", res)
	}
	if len(res.MissingEvidence) != 1 || res.MissingEvidence[0] != "when_evaluation" {
		t.Fatalf("missing_evidence: %v", res.MissingEvidence)
	}
}

func TestResolveRoute_resolved(t *testing.T) {
	t.Parallel()
	// Given: one matching route rule
	rules := []config.Rule{
		{Type: "route", ID: "r1", Priority: 10, RouteID: "next", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
	}
	// When: ResolveRoute runs
	res, err := ResolveRoute(rules, map[string]any{"x": 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Then: resolved with single candidate
	if res.Kind != OutcomeResolved || res.RouteID != "next" {
		t.Fatalf("%+v", res)
	}
	if len(res.RouteCandidates) != 1 || res.RouteCandidates[0].RouteID != "next" {
		t.Fatalf("route_candidates: %+v", res.RouteCandidates)
	}
}

func TestResolveRoute_orderedCandidates(t *testing.T) {
	t.Parallel()
	// Given: two matching routes at different priorities
	rules := []config.Rule{
		{Type: "route", ID: "low", Priority: 5, RouteID: "R5", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
		{Type: "route", ID: "high", Priority: 20, RouteID: "R20", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
	}
	// When: ResolveRoute runs
	res, err := ResolveRoute(rules, map[string]any{"x": 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Then: lower priority wins; candidates ordered by priority
	if res.RouteID != "R5" {
		t.Fatalf("want winner R5 got %+v", res)
	}
	if len(res.RouteCandidates) != 2 || res.RouteCandidates[0].RouteID != "R5" || res.RouteCandidates[1].RouteID != "R20" {
		t.Fatalf("route_candidates order: %+v", res.RouteCandidates)
	}
}

func TestResolveRoute_noMatchDegraded(t *testing.T) {
	t.Parallel()
	// Given: route rules that do not match
	// When: ResolveRoute runs
	res, err := ResolveRoute([]config.Rule{
		{Type: "route", ID: "r", Priority: 1, RouteID: "x", When: map[string]any{"op": "eq", "path": "a", "value": 1}},
	}, map[string]any{"a": 2}, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Then: degraded
	if res.Kind != OutcomeDegraded || !strings.Contains(res.Reason, "no matching route rule") {
		t.Fatalf("%+v", res)
	}
}

func TestResolveRoute_allSuppressed(t *testing.T) {
	t.Parallel()
	// Given: matching route suppressed by degraded dependency
	// When: ResolveRoute runs
	res, err := ResolveRoute([]config.Rule{
		{Type: "route", ID: "r", Priority: 1, RouteID: "x", When: map[string]any{"op": "eq", "path": "a", "value": 1}, DependsOn: []string{"git"}},
	}, map[string]any{"a": 1}, map[string]struct{}{"git": {}})
	if err != nil {
		t.Fatal(err)
	}
	// Then: degraded — all candidates suppressed
	if res.Kind != OutcomeDegraded || !strings.Contains(res.Reason, "suppressed") {
		t.Fatalf("%+v", res)
	}
}

func TestResolveRoute_missingRouteID(t *testing.T) {
	t.Parallel()
	// Given: route rule with empty route_id
	// When: ResolveRoute runs
	res, err := ResolveRoute([]config.Rule{
		{Type: "route", ID: "bad", Priority: 1, RouteID: "", When: map[string]any{"op": "eq", "path": "a", "value": 1}},
	}, map[string]any{"a": 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Then: unsupported outcome
	if res.Kind != OutcomeUnsupported || !strings.Contains(res.Reason, `route rule "bad" is missing route_id`) {
		t.Fatalf("%+v", res)
	}
}

func TestResolveRoute_winnerMissingRouteID_preservesRouteCandidates(t *testing.T) {
	t.Parallel()
	// Given: best-priority match has empty route_id but a higher-priority-number match has a valid route_id
	rules := []config.Rule{
		{Type: "route", ID: "bad", Priority: 5, RouteID: "", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
		{Type: "route", ID: "ok", Priority: 10, RouteID: "R10", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
	}
	// When: ResolveRoute runs
	res, err := ResolveRoute(rules, map[string]any{"x": 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Then: unsupported on winner, but ordered alternatives with non-empty route_id are still attached
	if res.Kind != OutcomeUnsupported {
		t.Fatalf("want unsupported, got %+v", res)
	}
	if len(res.RouteCandidates) != 1 || res.RouteCandidates[0].RouteID != "R10" {
		t.Fatalf("route_candidates: %+v", res.RouteCandidates)
	}
}

func TestResolveRoute_tieAmbiguous(t *testing.T) {
	t.Parallel()
	// Given: two matching routes at same priority
	// When: ResolveRoute runs
	res, err := ResolveRoute([]config.Rule{
		{Type: "route", ID: "a", Priority: 5, RouteID: "r1", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
		{Type: "route", ID: "b", Priority: 5, RouteID: "r2", When: map[string]any{"op": "eq", "path": "x", "value": 1}},
	}, map[string]any{"x": 1}, nil)
	if err != nil {
		t.Fatal(err)
	}
	// Then: ambiguous
	if res.Kind != OutcomeAmbiguous {
		t.Fatalf("%+v", res)
	}
}

func TestDuplicatePriorityWarnings(t *testing.T) {
	t.Parallel()
	// Given: two state rules sharing priority
	rules := []config.Rule{
		{Type: "state", ID: "a", Priority: 1, When: map[string]any{}},
		{Type: "state", ID: "b", Priority: 1, When: map[string]any{}},
	}
	// When: DuplicatePriorityWarnings runs
	w := DuplicatePriorityWarnings(rules)
	// Then: one warning
	if len(w) != 1 || !strings.Contains(w[0], "duplicated") {
		t.Fatalf("%v", w)
	}
}

func TestDuplicatePriorityWarnings_singleRuleSilent(t *testing.T) {
	t.Parallel()
	// Given: single rule
	// When: DuplicatePriorityWarnings runs
	w := DuplicatePriorityWarnings([]config.Rule{
		{Type: "state", ID: "only", Priority: 1, When: map[string]any{}},
	})
	// Then: no warnings
	if len(w) != 0 {
		t.Fatal(w)
	}
}

func TestDuplicatePriorityWarnings_separateTypes(t *testing.T) {
	t.Parallel()
	// Given: duplicate priorities within state and within route
	// When: DuplicatePriorityWarnings runs
	w := DuplicatePriorityWarnings([]config.Rule{
		{Type: "state", ID: "s1", Priority: 1, When: map[string]any{}},
		{Type: "state", ID: "s2", Priority: 1, When: map[string]any{}},
		{Type: "route", ID: "r1", Priority: 1, When: map[string]any{}},
		{Type: "route", ID: "r2", Priority: 1, When: map[string]any{}},
	})
	// Then: two within-type duplicate warnings plus one cross-kind warning for the shared priority.
	if len(w) != 3 {
		t.Fatalf("%v", w)
	}
}

func TestDuplicatePriorityWarnings_crossKindOnly(t *testing.T) {
	t.Parallel()
	// Given: one state and one route at same priority (no within-type duplicate)
	// When: DuplicatePriorityWarnings runs
	w := DuplicatePriorityWarnings([]config.Rule{
		{Type: "state", ID: "s1", Priority: 1, When: map[string]any{}},
		{Type: "route", ID: "r1", Priority: 1, When: map[string]any{}},
	})
	// Then: single cross-kind warning
	if len(w) != 1 || !strings.Contains(w[0], "shared across rule kinds") {
		t.Fatalf("%v", w)
	}
}
