// Package resolve applies ADR-0004 priority selection and ADR-0007 outcomes to control rules.
//
// # Inputs and outputs
//
// Callers pass loaded config.Rule values filtered to type "state", "route", or "guard", a flattened
// observation signal map (see package signals), and a set of degraded provider IDs for
// depends_on suppression. Resolve and ResolveState/ResolveRoute/ResolveGuard return Result with Kind
// resolved, ambiguous, degraded, or unsupported; TargetID mirrors the winning state_id,
// route_id, or guard_id when resolved. Route rules additionally populate RouteCandidates for ordered alternatives.
//
// Guard rules are only meaningful within a single guard_id: pass rules already scoped to one id
// (as [ResolveGuard] does) or ensure the slice contains only one guard_id before calling
// Resolve(..., "guard").
//
// # Error semantics
//
// Invalid ruleType, match evaluation errors, or missing state_id/route_id/guard_id on the single
// winning rule yield OutcomeUnsupported with Reason, MissingEvidence, and ReEntryHint (ADR-0007).
// No matching rules, all rules suppressed, or duplicate best priority return OutcomeDegraded or
// OutcomeAmbiguous with a Reason string. Resolve returns a nil error; callers inspect Result.Kind.
//
// ADR-0004 (priority and selection), ADR-0007 (outcomes and reporting), ADR-0013 (workflow state/route catalog).
package resolve

import (
	"fmt"
	"math"
	"sort"
	"strconv"

	"github.com/k-shibuki/reinguard/internal/config"
	"github.com/k-shibuki/reinguard/internal/match"
)

// priorityEpsilon is used when comparing rule priorities for ties.
const priorityEpsilon = 1e-9

// OutcomeKind classifies resolution result.
type OutcomeKind string

// Standard values for Result.Kind in control rule resolution (ADR-0007).
const (
	OutcomeResolved    OutcomeKind = "resolved"
	OutcomeAmbiguous   OutcomeKind = "ambiguous"
	OutcomeDegraded    OutcomeKind = "degraded"
	OutcomeUnsupported OutcomeKind = "unsupported"
)

// Result is the outcome of resolving among matching rules of one type.
type Result struct {
	Kind        OutcomeKind `json:"kind"`
	StateID     string      `json:"state_id,omitempty"`
	RouteID     string      `json:"route_id,omitempty"`
	GuardID     string      `json:"guard_id,omitempty"`
	TargetID    string      `json:"target_id,omitempty"`
	RuleID      string      `json:"rule_id,omitempty"`
	Reason      string      `json:"reason,omitempty"`
	ReEntryHint string      `json:"re_entry_hint,omitempty"`
	// RuleTrace is the optional rule-level evaluation trace; populated only by *Trace
	// entry points so default Result JSON stays unchanged. See RuleTrace.
	RuleTrace       []RuleTrace      `json:"rule_trace,omitempty"`
	Candidates      []string         `json:"candidates,omitempty"`
	RouteCandidates []RouteCandidate `json:"route_candidates,omitempty"`
	MissingEvidence []string         `json:"missing_evidence,omitempty"`
	Priority        float64          `json:"priority,omitempty"`
}

// RouteCandidate is one matching route rule after depends_on suppression, ordered for output.
type RouteCandidate struct {
	RuleID   string  `json:"rule_id"`
	RouteID  string  `json:"route_id"`
	Priority float64 `json:"priority"`
}

// RuleTrace is one rule-level entry in an optional resolution trace (Issue #143). It records
// rule_id, rule_type, priority, target_id (state_id / route_id / guard_id depending on rule_type),
// whether the rule's when-clause matched, and dependency suppression for matched rules.
//
// Trace entries are recorded for every rule of the requested type that the resolver evaluated,
// in evaluation order. They do not include the "and" / "or" / "any" / "all" sub-clauses of
// match expressions; rule_trace is intentionally rule-level only (no expression-level tracing).
//
// Suppressed and SuppressedBy are populated only for rules whose when-clause matched but whose
// declared depends_on entries include at least one degraded provider. SuppressedBy lists the
// degraded provider IDs that caused the suppression.
type RuleTrace struct {
	RuleID       string   `json:"rule_id"`
	RuleType     string   `json:"rule_type"`
	TargetID     string   `json:"target_id"`
	SuppressedBy []string `json:"suppressed_by,omitempty"`
	Priority     float64  `json:"priority"`
	Matched      bool     `json:"matched"`
	Suppressed   bool     `json:"suppressed,omitempty"`
}

// StateRules filters rules to type state.
func StateRules(rules []config.Rule) []config.Rule {
	return filterType(rules, "state")
}

// RouteRules filters rules to type route.
func RouteRules(rules []config.Rule) []config.Rule {
	return filterType(rules, "route")
}

func filterType(rules []config.Rule, typ string) []config.Rule {
	var out []config.Rule
	for _, r := range rules {
		if r.Type == typ {
			out = append(out, r)
		}
	}
	return out
}

// resolveProfile carries human-readable reasons and flags for one invocation of Resolve.
type resolveProfile struct {
	ruleType            string
	noMatchReason       string
	allSuppressedReason string
	ambiguousReason     string
	routeStyle          bool
}

// profileForRuleType returns the outcome strings and route-style flag for state, route, or guard rules.
func profileForRuleType(ruleType string) (resolveProfile, error) {
	switch ruleType {
	case "state":
		return resolveProfile{
			ruleType:            "state",
			noMatchReason:       "no matching state rule",
			allSuppressedReason: "all matches suppressed by depends_on",
			ambiguousReason:     "multiple rules at same best priority",
			routeStyle:          false,
		}, nil
	case "route":
		return resolveProfile{
			ruleType:            "route",
			noMatchReason:       "no matching route rule",
			allSuppressedReason: "all route matches suppressed",
			ambiguousReason:     "multiple route rules at same best priority",
			routeStyle:          true,
		}, nil
	case "guard":
		return resolveProfile{
			ruleType:            "guard",
			noMatchReason:       "no matching guard rule",
			allSuppressedReason: "all guard matches suppressed by depends_on",
			ambiguousReason:     "multiple guard rules at same best priority",
			routeStyle:          false,
		}, nil
	default:
		return resolveProfile{}, fmt.Errorf("unsupported rule type %q", ruleType)
	}
}

func unsupportedRuleType(ruleType, profileErr string) Result {
	return Result{
		Kind:            OutcomeUnsupported,
		Reason:          fmt.Sprintf("resolve: %s", profileErr),
		MissingEvidence: []string{fmt.Sprintf("rule_type:%s", ruleType)},
		ReEntryHint:     `Pass rule_type "state", "route", or "guard" only.`,
	}
}

func unsupportedMatchError(cause error) Result {
	return Result{
		Kind:            OutcomeUnsupported,
		Reason:          fmt.Sprintf("resolve: %v", cause),
		MissingEvidence: []string{"when_evaluation"},
		ReEntryHint:     "Fix the failing rule's when-clause (see reason) or supply observation values it requires; see ADR-0002.",
	}
}

func unsupportedMissingStateID(ruleID string) Result {
	return Result{
		Kind:            OutcomeUnsupported,
		Reason:          fmt.Sprintf("state rule %q is missing state_id", ruleID),
		MissingEvidence: []string{fmt.Sprintf("rule_id:%s", ruleID), "state_id"},
		ReEntryHint:     "Set state_id on the rule under control/states and run rgd config validate.",
	}
}

func unsupportedMissingRouteID(ruleID string) Result {
	return Result{
		Kind:            OutcomeUnsupported,
		Reason:          fmt.Sprintf("route rule %q is missing route_id", ruleID),
		MissingEvidence: []string{fmt.Sprintf("rule_id:%s", ruleID), "route_id"},
		ReEntryHint:     "Set route_id on the rule under control/routes and run rgd config validate.",
	}
}

func unsupportedMissingGuardID(ruleID string) Result {
	return Result{
		Kind:            OutcomeUnsupported,
		Reason:          fmt.Sprintf("guard rule %q is missing guard_id", ruleID),
		MissingEvidence: []string{fmt.Sprintf("rule_id:%s", ruleID), "guard_id"},
		ReEntryHint:     "Set guard_id on the rule under control/guards and run rgd config validate.",
	}
}

// Resolve applies ADR-0004 selection to rules of a single type ("state", "route", or "guard"). On success
// Kind is OutcomeResolved; otherwise Kind may be OutcomeDegraded, OutcomeAmbiguous, or
// OutcomeUnsupported (ADR-0007) with Reason and optional handoff fields.
//
// For ruleType "guard", rules must share one logical guard_id (or use [ResolveGuard], which filters).
//
// Resolve never populates Result.RuleTrace; use [ResolveTrace] when callers need a rule-level
// evaluation trace (Issue #143). Resolve and ResolveTrace agree on every other field.
func Resolve(rules []config.Rule, signals map[string]any, degraded map[string]struct{}, ruleType string) (Result, error) {
	return resolveCore(rules, signals, degraded, ruleType, false)
}

// ResolveTrace is the trace-aware variant of [Resolve]: in addition to the standard Result fields,
// it populates Result.RuleTrace with one entry per evaluated rule of the requested type, in
// evaluation order, including non-matching rules and matched-but-suppressed rules.
//
// Trace coverage matches [Resolve]'s evaluation: rules whose when-clause panics with an evaluation
// error stop the walk and are reported as OutcomeUnsupported (the failing rule appears in
// Result.Reason / Result.MissingEvidence). Rules evaluated before that point still appear in
// Result.RuleTrace; the failing rule itself is not appended because evaluation never finished.
func ResolveTrace(rules []config.Rule, signals map[string]any, degraded map[string]struct{}, ruleType string) (Result, error) {
	return resolveCore(rules, signals, degraded, ruleType, true)
}

func resolveCore(rules []config.Rule, signals map[string]any, degraded map[string]struct{}, ruleType string, withTrace bool) (Result, error) {
	p, err := profileForRuleType(ruleType)
	if err != nil {
		return unsupportedRuleType(ruleType, err.Error()), nil
	}
	evals, evalErr := evaluateRules(filterType(rules, p.ruleType), signals, degraded)
	if evalErr != nil {
		res := unsupportedMatchError(evalErr)
		if withTrace {
			res.RuleTrace = traceFromEvals(evals)
		}
		return res, nil
	}
	candidates, active := splitMatchedRules(evals)
	res := selectFromActive(p, candidates, active)
	if withTrace {
		res.RuleTrace = traceFromEvals(evals)
	}
	return res, nil
}

// ruleEval captures one rule's evaluation outcome so the same data can drive both selection and
// optional rule-level tracing without re-walking match.Eval.
type ruleEval struct {
	suppressedBy []string
	rule         config.Rule
	matched      bool
}

func selectFromActive(p resolveProfile, candidates, active []config.Rule) Result {
	if len(candidates) == 0 {
		return Result{Kind: OutcomeDegraded, Reason: p.noMatchReason}
	}
	if len(active) == 0 {
		return Result{Kind: OutcomeDegraded, Reason: p.allSuppressedReason}
	}
	ordered, routeCandidates := orderRulesForResolve(active, p.routeStyle)
	best := minPriority(ordered)
	var atBest []config.Rule
	for _, c := range ordered {
		if nearlyEqual(c.Priority, best) {
			atBest = append(atBest, c)
		}
	}
	if len(atBest) > 1 {
		return ambiguousResolveResult(best, atBest, p, routeCandidates)
	}
	return singleRuleResolveResult(atBest[0], p, routeCandidates)
}

// orderRulesForResolve returns rules in evaluation order; for routes, copies, sorts, and builds routeCandidates.
func orderRulesForResolve(active []config.Rule, routeStyle bool) (ordered []config.Rule, routeCandidates []RouteCandidate) {
	ordered = active
	if !routeStyle {
		return ordered, nil
	}
	ordered = append([]config.Rule(nil), active...)
	sort.Slice(ordered, func(i, j int) bool {
		if ordered[i].Priority != ordered[j].Priority {
			return ordered[i].Priority < ordered[j].Priority
		}
		return ordered[i].ID < ordered[j].ID
	})
	for _, c := range ordered {
		if c.RouteID == "" {
			continue
		}
		routeCandidates = append(routeCandidates, RouteCandidate{
			RuleID:   c.ID,
			RouteID:  c.RouteID,
			Priority: c.Priority,
		})
	}
	return ordered, routeCandidates
}

func ambiguousResolveResult(best float64, atBest []config.Rule, p resolveProfile, routeCandidates []RouteCandidate) Result {
	ids := make([]string, len(atBest))
	for i := range atBest {
		ids[i] = atBest[i].ID
	}
	return Result{
		Kind:            OutcomeAmbiguous,
		Priority:        best,
		Candidates:      ids,
		RouteCandidates: routeCandidates,
		Reason:          p.ambiguousReason,
	}
}

func singleRuleResolveResult(r config.Rule, p resolveProfile, routeCandidates []RouteCandidate) Result {
	if p.routeStyle {
		if r.RouteID == "" {
			res := unsupportedMissingRouteID(r.ID)
			res.RouteCandidates = routeCandidates
			return res
		}
		return Result{
			Kind:            OutcomeResolved,
			RouteID:         r.RouteID,
			TargetID:        r.RouteID,
			RuleID:          r.ID,
			Priority:        r.Priority,
			RouteCandidates: routeCandidates,
		}
	}
	if p.ruleType == "guard" {
		if r.GuardID == "" {
			return unsupportedMissingGuardID(r.ID)
		}
		return Result{
			Kind:     OutcomeResolved,
			GuardID:  r.GuardID,
			TargetID: r.GuardID,
			RuleID:   r.ID,
			Priority: r.Priority,
		}
	}
	if r.StateID == "" {
		return unsupportedMissingStateID(r.ID)
	}
	return Result{
		Kind:     OutcomeResolved,
		StateID:  r.StateID,
		TargetID: r.StateID,
		RuleID:   r.ID,
		Priority: r.Priority,
	}
}

// minPriority returns the minimum priority in rules. rules must be non-empty;
// Resolve only calls this after len(active) > 0.
func minPriority(rules []config.Rule) float64 {
	best := rules[0].Priority
	for _, c := range rules[1:] {
		if c.Priority < best {
			best = c.Priority
		}
	}
	return best
}

// ResolveState evaluates state rules against signals and degraded sources (convenience wrapper for Resolve(..., "state")).
func ResolveState(rules []config.Rule, signals map[string]any, degraded map[string]struct{}) (Result, error) {
	return Resolve(rules, signals, degraded, "state")
}

// ResolveStateTrace is the trace-aware variant of [ResolveState] (Issue #143).
func ResolveStateTrace(rules []config.Rule, signals map[string]any, degraded map[string]struct{}) (Result, error) {
	return ResolveTrace(rules, signals, degraded, "state")
}

// ResolveRoute selects route rules with the same priority and depends_on semantics as state rules.
func ResolveRoute(rules []config.Rule, signals map[string]any, degraded map[string]struct{}) (Result, error) {
	return Resolve(rules, signals, degraded, "route")
}

// ResolveRouteTrace is the trace-aware variant of [ResolveRoute] (Issue #143).
func ResolveRouteTrace(rules []config.Rule, signals map[string]any, degraded map[string]struct{}) (Result, error) {
	return ResolveTrace(rules, signals, degraded, "route")
}

// ResolveGuard selects among guard rules that declare the given guard_id using the same priority
// and depends_on semantics as ResolveState (ADR-0004). It is the supported entry point for guard
// resolution; callers should not mix multiple guard_id values in one Resolve(..., "guard") call.
func ResolveGuard(rules []config.Rule, signals map[string]any, degraded map[string]struct{}, guardID string) (Result, error) {
	return Resolve(scopeGuardRules(rules, guardID), signals, degraded, "guard")
}

// ResolveGuardTrace is the trace-aware variant of [ResolveGuard] (Issue #143). The trace covers
// only rules scoped to guardID; rules with a different guard_id are filtered before evaluation
// and never appear in Result.RuleTrace.
func ResolveGuardTrace(rules []config.Rule, signals map[string]any, degraded map[string]struct{}, guardID string) (Result, error) {
	return ResolveTrace(scopeGuardRules(rules, guardID), signals, degraded, "guard")
}

func scopeGuardRules(rules []config.Rule, guardID string) []config.Rule {
	var scoped []config.Rule
	for _, r := range rules {
		if r.Type == "guard" && r.GuardID == guardID {
			scoped = append(scoped, r)
		}
	}
	return scoped
}

// evaluateRules walks the typed rule slice once, recording match.Eval outcome and depends_on
// suppression per rule. The first match.Eval failure aborts the walk and returns the rules
// evaluated so far together with the wrapped error.
func evaluateRules(rules []config.Rule, signals map[string]any, degraded map[string]struct{}) ([]ruleEval, error) {
	out := make([]ruleEval, 0, len(rules))
	for _, r := range rules {
		ok, err := match.Eval(r.When, signals)
		if err != nil {
			return out, fmt.Errorf("rule %s: %w", r.ID, err)
		}
		re := ruleEval{rule: r, matched: ok}
		if ok && len(degraded) > 0 {
			for _, d := range r.DependsOn {
				if _, isDeg := degraded[d]; isDeg {
					re.suppressedBy = append(re.suppressedBy, d)
				}
			}
		}
		out = append(out, re)
	}
	return out, nil
}

// splitMatchedRules separates evaluated rules into (matched, matched-and-not-suppressed) preserving
// the original evaluation order.
func splitMatchedRules(evals []ruleEval) (candidates, active []config.Rule) {
	for _, e := range evals {
		if !e.matched {
			continue
		}
		candidates = append(candidates, e.rule)
		if len(e.suppressedBy) == 0 {
			active = append(active, e.rule)
		}
	}
	return candidates, active
}

// traceFromEvals projects ruleEval entries into the public RuleTrace shape. Suppressed and
// SuppressedBy are populated only for matched rules with degraded dependencies.
func traceFromEvals(evals []ruleEval) []RuleTrace {
	if len(evals) == 0 {
		return nil
	}
	out := make([]RuleTrace, 0, len(evals))
	for _, e := range evals {
		entry := RuleTrace{
			RuleID:   e.rule.ID,
			RuleType: e.rule.Type,
			Priority: e.rule.Priority,
			TargetID: ruleTargetID(e.rule),
			Matched:  e.matched,
		}
		if len(e.suppressedBy) > 0 {
			entry.Suppressed = true
			entry.SuppressedBy = append([]string(nil), e.suppressedBy...)
		}
		out = append(out, entry)
	}
	return out
}

// ruleTargetID returns the rule's logical target identifier (state_id / route_id / guard_id) per
// rule type. Empty strings are preserved so callers can distinguish "no target" from "missing".
func ruleTargetID(r config.Rule) string {
	switch r.Type {
	case "state":
		return r.StateID
	case "route":
		return r.RouteID
	case "guard":
		return r.GuardID
	default:
		return ""
	}
}

// NearlyEqual reports floating-point equality for duplicate-priority warnings (ADR-0004).
func NearlyEqual(a, b float64) bool {
	return nearlyEqual(a, b)
}

func nearlyEqual(a, b float64) bool {
	return math.Abs(a-b) <= priorityEpsilon
}

// DuplicatePriorityWarnings returns validation warnings for rules that share the same
// numeric priority (ADR-0004: one shared priority space across rule kinds).
func DuplicatePriorityWarnings(rules []config.Rule) []string {
	by := map[string]map[string][]string{} // type -> priorityKey -> ids
	for _, r := range rules {
		if _, ok := by[r.Type]; !ok {
			by[r.Type] = map[string][]string{}
		}
		key := strconv.FormatFloat(r.Priority, 'f', -1, 64)
		by[r.Type][key] = append(by[r.Type][key], r.ID)
	}
	var msgs []string
	for typ, m := range by {
		for priKey, ids := range m {
			if len(ids) < 2 {
				continue
			}
			msgs = append(msgs, fmt.Sprintf("type %s priority %s duplicated by rules %v", typ, priKey, ids))
		}
	}

	// Cross-kind collisions: same numeric priority on different rule types (not caught above
	// when each type has at most one rule at that priority).
	global := map[string][]config.Rule{} // priorityKey -> rules
	for _, r := range rules {
		key := strconv.FormatFloat(r.Priority, 'f', -1, 64)
		global[key] = append(global[key], r)
	}
	for priKey, group := range global {
		if len(group) < 2 {
			continue
		}
		typeSeen := map[string]struct{}{}
		for _, r := range group {
			typeSeen[r.Type] = struct{}{}
		}
		if len(typeSeen) < 2 {
			continue
		}
		ids := make([]string, len(group))
		for i := range group {
			ids[i] = group[i].ID
		}
		msgs = append(msgs, fmt.Sprintf("priority %s shared across rule kinds by rules %v", priKey, ids))
	}
	sort.Strings(msgs)
	return msgs
}
