package rgdcli

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

var updateGolden = flag.Bool("update", false, "refresh rgd CLI golden fixtures")

func TestGolden_workflowStateAndRoute(t *testing.T) {
	t.Parallel()
	cfgDir := goldenWorkflowConfigDir(t)
	for _, tc := range goldenWorkflowCases() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			obsPath := goldenObservationPath(t, "state_eval", tc.name, tc.observation)
			stateOut := runGoldenCommand(t, "rgd", "state", "eval", "--config-dir", cfgDir, "--observation-file", obsPath)
			assertGoldenOutput(t, "state_eval", tc.name, stateOut)
			assertJSONPathString(t, stateOut, "state_id", tc.stateID)

			stateFile := filepath.Join(t.TempDir(), "state.json")
			writeFile(t, stateFile, stateOut)
			routeObsPath := goldenObservationPath(t, "route_select", tc.name, tc.observation)
			routeOut := runGoldenCommand(t, "rgd", "route", "select", "--config-dir", cfgDir, "--observation-file", routeObsPath, "--state-file", stateFile)
			assertGoldenOutput(t, "route_select", tc.name, routeOut)
			assertJSONPathString(t, routeOut, "route_id", tc.routeID)
		})
	}
}

func TestGolden_mergeReadinessGuard(t *testing.T) {
	t.Parallel()
	cfgDir := goldenWorkflowConfigDir(t)
	for _, tc := range goldenGuardCases() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			obsPath := goldenObservationPath(t, "guard_eval", tc.name, tc.observation)
			out := runGoldenCommand(t, "rgd", "guard", "eval", "--config-dir", cfgDir, "--observation-file", obsPath, "merge-readiness")
			assertGoldenOutput(t, "guard_eval", tc.name, out)
			assertJSONPathBool(t, out, "ok", tc.ok)
		})
	}
}

func TestGolden_contextBuild(t *testing.T) {
	t.Parallel()
	cfgDir := goldenWorkflowConfigDir(t)
	for _, tc := range goldenContextCases() {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			obsPath := goldenObservationPath(t, "context_build", tc.name, tc.observation)
			args := []string{"rgd", "context", "build", "--config-dir", cfgDir, "--observation-file", obsPath}
			args = append(args, tc.args...)
			out := runGoldenCommand(t, args...)
			assertGoldenOutput(t, "context_build", tc.name, out)
			assertJSONPathString(t, out, "state.state_id", tc.stateID)
			if !tc.traceRules {
				assertJSONPathMissing(t, out, "state.rule_trace")
				assertJSONPathMissing(t, out, "routes.0.rule_trace")
			}
		})
	}
}

func TestGoldenCoverage_FSMStateIdsHaveFixture(t *testing.T) {
	t.Parallel()
	want := workflowCaseStateIDs()
	got := readWorkflowStateIDs(t)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("workflow state fixture coverage mismatch\ngot:  %v\nwant: %v", got, want)
	}
}

func TestGoldenCoverage_RouteIdsHaveFixture(t *testing.T) {
	t.Parallel()
	want := workflowCaseRouteIDs()
	got := readWorkflowRouteIDs(t)
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("workflow route fixture coverage mismatch\ngot:  %v\nwant: %v", got, want)
	}
}

func TestGoldenCoverage_MergeReadinessGuardHasFixture(t *testing.T) {
	t.Parallel()
	guards := readWorkflowGuardIDs(t)
	if !containsString(guards, "merge-readiness") {
		t.Fatalf("test bug: committed guard ids did not include merge-readiness: %v", guards)
	}
	names := map[string]bool{}
	for _, tc := range goldenGuardCases() {
		names[tc.name] = true
	}
	for _, name := range []string{"pass_merge_ready", "fail_unresolved_threads", "fail_changes_requested", "fail_ci_not_success", "fail_trigger_awaiting_ack"} {
		if !names[name] {
			t.Fatalf("merge-readiness guard fixture %q missing", name)
		}
	}
}

func goldenCaseDir(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	return filepath.Join(filepath.Dir(file), "testdata", "golden", name)
}

func goldenRepoRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func goldenWorkflowConfigDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	repo := goldenRepoRoot(t)
	for _, src := range []string{
		".reinguard/control/states/workflow.yaml",
		".reinguard/control/routes/workflow.yaml",
		".reinguard/control/guards/default.yaml",
	} {
		writeFile(t, filepath.Join(root, strings.TrimPrefix(src, ".reinguard/")), readFile(t, filepath.Join(repo, src)))
	}
	if err := os.Mkdir(filepath.Join(root, "knowledge"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, "knowledge", "manifest.json"), []byte(`{"schema_version":"0.8.0","entries":[]}`))
	return root
}

func readFile(t *testing.T, p string) []byte {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

func runGoldenCommand(t *testing.T, args ...string) []byte {
	t.Helper()
	var buf bytes.Buffer
	app := NewApp("test")
	app.Writer = &buf
	app.ErrWriter = &bytes.Buffer{}
	if err := app.Run(args); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func goldenObservationPath(t *testing.T, suite, name string, observation map[string]any) string {
	t.Helper()
	dir := filepath.Join(goldenCaseDir(t, suite), name)
	path := filepath.Join(dir, "observation.json")
	if *updateGolden {
		writeCanonicalJSONFile(t, path, observation)
	}
	return path
}

func assertGoldenOutput(t *testing.T, suite, name string, got []byte) {
	t.Helper()
	wantPath := filepath.Join(goldenCaseDir(t, suite), name, "want.json")
	if *updateGolden {
		writeCanonicalJSONFile(t, wantPath, mustDecodeJSON(t, got))
		return
	}
	assertCanonicalJSONEqual(t, got, readFile(t, wantPath))
}

func writeCanonicalJSONFile(t *testing.T, path string, v any) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	b, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	b = append(b, '\n')
	if err := os.WriteFile(path, b, 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertCanonicalJSONEqual(t *testing.T, gotJSON, wantJSON []byte) {
	t.Helper()
	var got, want any
	if err := json.Unmarshal(gotJSON, &got); err != nil {
		t.Fatalf("got: %v", err)
	}
	if err := json.Unmarshal(wantJSON, &want); err != nil {
		t.Fatalf("want: %v", err)
	}
	gb, err := json.Marshal(got)
	if err != nil {
		t.Fatal(err)
	}
	wb, err := json.Marshal(want)
	if err != nil {
		t.Fatal(err)
	}
	if string(gb) != string(wb) {
		t.Fatalf("JSON mismatch\ngot:  %s\nwant: %s", gb, wb)
	}
}

func mustDecodeJSON(t *testing.T, data []byte) any {
	t.Helper()
	var out any
	if err := json.Unmarshal(data, &out); err != nil {
		t.Fatal(err)
	}
	return out
}

func assertJSONPathString(t *testing.T, data []byte, path, want string) {
	t.Helper()
	got, ok := jsonPath(t, data, path).(string)
	if !ok || got != want {
		t.Fatalf("%s = %#v, want %q", path, got, want)
	}
}

func assertJSONPathBool(t *testing.T, data []byte, path string, want bool) {
	t.Helper()
	got, ok := jsonPath(t, data, path).(bool)
	if !ok || got != want {
		t.Fatalf("%s = %#v, want %v", path, got, want)
	}
}

func assertJSONPathMissing(t *testing.T, data []byte, path string) {
	t.Helper()
	if got := jsonPath(t, data, path); got != nil {
		t.Fatalf("%s = %#v, want missing", path, got)
	}
}

func jsonPath(t *testing.T, data []byte, path string) any {
	t.Helper()
	cur := mustDecodeJSON(t, data)
	for _, part := range strings.Split(path, ".") {
		switch x := cur.(type) {
		case map[string]any:
			var ok bool
			cur, ok = x[part]
			if !ok {
				return nil
			}
		case []any:
			var idx int
			if _, err := fmt.Sscanf(part, "%d", &idx); err != nil {
				t.Fatalf("invalid path index %q in %s", part, path)
			}
			if idx < 0 || idx >= len(x) {
				return nil
			}
			cur = x[idx]
		default:
			return nil
		}
	}
	return cur
}

//nolint:govet // Test case readability is more important than fieldalignment packing here.
type goldenWorkflowCase struct {
	name        string
	stateID     string
	routeID     string
	observation map[string]any
}

//nolint:govet // Test case readability is more important than fieldalignment packing here.
type goldenGuardCase struct {
	name        string
	ok          bool
	observation map[string]any
}

//nolint:govet // Test case readability is more important than fieldalignment packing here.
type goldenContextCase struct {
	name        string
	stateID     string
	args        []string
	traceRules  bool
	observation map[string]any
}

func goldenWorkflowCases() []goldenWorkflowCase {
	return []goldenWorkflowCase{
		{"working_no_pr", "working_no_pr", "user-implement", fixtureObservation(baseSignals(noPR()))},
		{"ready_for_pr", "ready_for_pr", "user-implement", fixtureObservation(withGates(baseSignals(noPR()), "pr-readiness", "pass"))},
		{"pr_open", "pr_open", "user-monitor-pr", fixtureObservation(baseSignals(prSignals()))},
		{"waiting_ci", "waiting_ci", "user-wait-ci", fixtureObservation(baseSignals(waitingCISignals()))},
		{"unresolved_threads", "unresolved_threads", "user-address-review", fixtureObservation(baseSignals(unresolvedThreadSignals()))},
		{"non_thread_findings_pending", "non_thread_findings_pending", "user-address-review", fixtureObservation(baseSignals(nonThreadSignals()))},
		{"changes_requested", "changes_requested", "user-address-review", fixtureObservation(baseSignals(changesRequestedSignals()))},
		{"waiting_bot_rate_limited", "waiting_bot_rate_limited", "user-wait-bot-quota", fixtureObservation(baseSignals(rateLimitedSignals()))},
		{"waiting_bot_paused", "waiting_bot_paused", "user-wait-bot-paused", fixtureObservation(baseSignals(pausedSignals()))},
		{"waiting_bot_failed", "waiting_bot_failed", "user-wait-bot-failed", fixtureObservation(baseSignals(failedSignals()))},
		{"waiting_bot_stale", "waiting_bot_stale", "user-wait-bot-stale", fixtureObservation(baseSignals(staleSignals()))},
		{"waiting_bot_run", "waiting_bot_run", "user-wait-bot-run", fixtureObservation(baseSignals(pendingBotSignals()))},
		{"merge_ready", "merge_ready", "user-merge", fixtureObservation(baseSignals(mergeReadySignals()))},
	}
}

func goldenGuardCases() []goldenGuardCase {
	pass := baseSignals(mergeReadySignals())
	return []goldenGuardCase{
		{"pass_merge_ready", true, fixtureObservation(pass)},
		{"fail_unresolved_threads", false, fixtureObservation(baseSignals(unresolvedThreadSignals()))},
		{"fail_changes_requested", false, fixtureObservation(baseSignals(changesRequestedSignals()))},
		{"fail_ci_not_success", false, fixtureObservation(baseSignals(ciFailureSignals()))},
		{"fail_trigger_awaiting_ack", false, fixtureObservation(baseSignals(triggerAwaitingAckSignals()))},
	}
}

func goldenContextCases() []goldenContextCase {
	return []goldenContextCase{
		{"working_no_pr", "working_no_pr", nil, false, fixtureObservation(baseSignals(noPR()))},
		{"merge_ready_full_green", "merge_ready", nil, false, fixtureObservation(baseSignals(mergeReadySignals()))},
		{"interaction_rate_limited_with_non_thread", "waiting_bot_rate_limited", nil, false, fixtureObservation(baseSignals(rateLimitedWithNonThreadSignals()))},
		{"interaction_paused_with_changes_requested", "waiting_bot_paused", nil, false, fixtureObservation(baseSignals(pausedWithChangesRequestedSignals()))},
		{"interaction_trigger_awaiting_ack_blocks_merge_ready", "waiting_bot_run", nil, false, fixtureObservation(baseSignals(triggerAwaitingAckSignals()))},
		{"interaction_priority_rate_limited_over_stale", "waiting_bot_rate_limited", nil, false, fixtureObservation(baseSignals(rateLimitedOverStaleSignals()))},
		{"compact_observation_view", "unresolved_threads", []string{"--compact"}, false, fixtureObservation(withHighVolumeReviewPayload(baseSignals(mergeReadySignals())))},
		{"trace_rules_enabled", "working_no_pr", []string{"--trace-rules"}, true, fixtureObservation(baseSignals(noPR()))},
		{"degraded_observation_warning", "working_no_pr", nil, false, degradedObservation(baseSignals(noPR()))},
	}
}

func fixtureObservation(signals map[string]any) map[string]any {
	return map[string]any{
		"schema_version": "0.8.0",
		"signals":        signals,
		"degraded":       false,
	}
}

func degradedObservation(signals map[string]any) map[string]any {
	obs := fixtureObservation(signals)
	obs["degraded"] = true
	obs["diagnostics"] = []any{map[string]any{
		"severity": "warning",
		"message":  "fixture provider degraded",
		"code":     "fixture_degraded",
		"provider": "fixture",
	}}
	return obs
}

func baseSignals(github map[string]any) map[string]any {
	return map[string]any{
		"git": map[string]any{
			"branch":                      "test/74-fsm-golden-fixtures-and-ci",
			"detached_head":               false,
			"uncommitted_files":           0,
			"working_tree_clean":          true,
			"stash_count":                 0,
			"ahead_of_upstream":           0,
			"behind_of_upstream":          0,
			"has_upstream":                false,
			"stale_remote_branches_count": 0,
		},
		"github": github,
	}
}

func noPR() map[string]any {
	return map[string]any{
		"pull_requests": map[string]any{
			"current_branch":       "test/74-fsm-golden-fixtures-and-ci",
			"open_count":           0,
			"pr_exists_for_branch": false,
			"pr_number_for_branch": 0,
		},
	}
}

func prSignals() map[string]any {
	return map[string]any{
		"pull_requests": map[string]any{
			"current_branch":       "test/74-fsm-golden-fixtures-and-ci",
			"open_count":           1,
			"pr_exists_for_branch": true,
			"pr_number_for_branch": 74,
			"merge_state_status":   "dirty",
		},
	}
}

func waitingCISignals() map[string]any {
	gh := mergeReadySignals()
	gh["ci"].(map[string]any)["ci_status"] = "pending"
	return gh
}

func mergeReadySignals() map[string]any {
	return map[string]any{
		"pull_requests": map[string]any{
			"current_branch":       "test/74-fsm-golden-fixtures-and-ci",
			"open_count":           1,
			"pr_exists_for_branch": true,
			"pr_number_for_branch": 74,
			"merge_state_status":   "clean",
		},
		"ci":      map[string]any{"ci_status": "success", "head_sha": "0123456789abcdef0123456789abcdef01234567"},
		"reviews": cleanReviews(),
	}
}

func cleanReviews() map[string]any {
	return map[string]any{
		"review_threads_total":               0,
		"review_threads_unresolved":          0,
		"pagination_incomplete":              false,
		"review_decisions_total":             1,
		"review_decisions_approved":          1,
		"review_decisions_changes_requested": 0,
		"review_decisions_truncated":         false,
		"bot_reviewer_status": []any{map[string]any{
			"id":                "coderabbit",
			"login":             "coderabbitai[bot]",
			"required":          true,
			"status":            "completed_clean",
			"has_review":        true,
			"review_state":      "APPROVED",
			"review_commit_sha": "0123456789abcdef0123456789abcdef01234567",
		}},
		"bot_review_diagnostics": map[string]any{
			"bot_review_completed":            true,
			"bot_review_pending":              false,
			"bot_review_blocked":              false,
			"bot_review_block_reason":         "",
			"bot_review_terminal":             true,
			"bot_review_failed":               false,
			"bot_review_stale":                false,
			"bot_review_trigger_awaiting_ack": false,
			"non_thread_findings_present":     false,
			"duplicate_findings_detected":     false,
		},
	}
}

func unresolvedThreadSignals() map[string]any {
	gh := mergeReadySignals()
	gh["reviews"].(map[string]any)["review_threads_total"] = 1
	gh["reviews"].(map[string]any)["review_threads_unresolved"] = 1
	return gh
}

func nonThreadSignals() map[string]any {
	gh := mergeReadySignals()
	gh["reviews"].(map[string]any)["bot_review_diagnostics"].(map[string]any)["non_thread_findings_present"] = true
	gh["reviews"].(map[string]any)["bot_reviewer_status"].([]any)[0].(map[string]any)["actionable_findings_count"] = 1
	gh["reviews"].(map[string]any)["bot_reviewer_status"].([]any)[0].(map[string]any)["status"] = "completed"
	return gh
}

func changesRequestedSignals() map[string]any {
	gh := mergeReadySignals()
	gh["reviews"].(map[string]any)["bot_reviewer_status"].([]any)[0].(map[string]any)["review_state"] = "CHANGES_REQUESTED"
	gh["reviews"].(map[string]any)["bot_reviewer_status"].([]any)[0].(map[string]any)["status"] = "completed"
	gh["reviews"].(map[string]any)["review_decisions_approved"] = 0
	gh["reviews"].(map[string]any)["review_decisions_changes_requested"] = 1
	return gh
}

func rateLimitedSignals() map[string]any {
	gh := prSignals()
	gh["reviews"] = withEmptyDecisionCounts(map[string]any{
		"bot_reviewer_status":    []any{map[string]any{"id": "coderabbit", "login": "coderabbitai[bot]", "required": true, "status": "rate_limited", "rate_limit_remaining_seconds": 120}},
		"bot_review_diagnostics": map[string]any{"bot_review_blocked": true, "bot_review_block_reason": "rate_limited"},
	})
	return gh
}

func pausedSignals() map[string]any {
	gh := prSignals()
	gh["reviews"] = withEmptyDecisionCounts(map[string]any{
		"bot_reviewer_status":    []any{map[string]any{"id": "coderabbit", "login": "coderabbitai[bot]", "required": true, "status": "review_paused"}},
		"bot_review_diagnostics": map[string]any{"bot_review_blocked": true, "bot_review_block_reason": "review_paused"},
	})
	return gh
}

func failedSignals() map[string]any {
	gh := prSignals()
	gh["reviews"] = withEmptyDecisionCounts(map[string]any{"bot_review_diagnostics": map[string]any{"bot_review_failed": true}})
	return gh
}

func staleSignals() map[string]any {
	gh := prSignals()
	gh["reviews"] = withEmptyDecisionCounts(map[string]any{"bot_review_diagnostics": map[string]any{"bot_review_stale": true}})
	return gh
}

func pendingBotSignals() map[string]any {
	gh := prSignals()
	gh["reviews"] = withEmptyDecisionCounts(map[string]any{"bot_review_diagnostics": map[string]any{"bot_review_pending": true}})
	return gh
}

func ciFailureSignals() map[string]any {
	gh := mergeReadySignals()
	gh["ci"].(map[string]any)["ci_status"] = "failure"
	return gh
}

func triggerAwaitingAckSignals() map[string]any {
	gh := mergeReadySignals()
	gh["reviews"].(map[string]any)["bot_reviewer_status"].([]any)[0].(map[string]any)["status"] = "pending"
	gh["reviews"].(map[string]any)["bot_reviewer_status"].([]any)[0].(map[string]any)["review_trigger_awaiting_ack"] = true
	diags := gh["reviews"].(map[string]any)["bot_review_diagnostics"].(map[string]any)
	diags["bot_review_completed"] = false
	diags["bot_review_pending"] = true
	diags["bot_review_terminal"] = false
	diags["bot_review_trigger_awaiting_ack"] = true
	return gh
}

func rateLimitedWithNonThreadSignals() map[string]any {
	gh := rateLimitedSignals()
	gh["reviews"].(map[string]any)["review_threads_unresolved"] = 0
	gh["reviews"].(map[string]any)["pagination_incomplete"] = false
	gh["reviews"].(map[string]any)["review_decisions_changes_requested"] = 0
	gh["reviews"].(map[string]any)["bot_review_diagnostics"].(map[string]any)["non_thread_findings_present"] = true
	return gh
}

func pausedWithChangesRequestedSignals() map[string]any {
	gh := pausedSignals()
	gh["reviews"].(map[string]any)["review_decisions_total"] = 1
	gh["reviews"].(map[string]any)["review_decisions_changes_requested"] = 1
	return gh
}

func withEmptyDecisionCounts(reviews map[string]any) map[string]any {
	reviews["review_decisions_total"] = 0
	reviews["review_decisions_approved"] = 0
	reviews["review_decisions_changes_requested"] = 0
	reviews["review_decisions_truncated"] = false
	return reviews
}

func rateLimitedOverStaleSignals() map[string]any {
	gh := rateLimitedSignals()
	gh["reviews"].(map[string]any)["bot_review_diagnostics"].(map[string]any)["bot_review_stale"] = true
	return gh
}

func withHighVolumeReviewPayload(signals map[string]any) map[string]any {
	gh := signals["github"].(map[string]any)
	gh["ci"].(map[string]any)["check_runs"] = []any{map[string]any{"name": "ci-pass", "status": "completed", "conclusion": "success"}}
	reviews := gh["reviews"].(map[string]any)
	reviews["review_inbox"] = []any{map[string]any{"thread_id": "thread-1", "root_comment_id": 1001, "path": "internal/rgdcli/golden_test.go", "line": 1}}
	reviews["review_threads_total"] = 1
	reviews["review_threads_unresolved"] = 1
	reviews["conversation_comments"] = []any{map[string]any{"author": "coderabbitai[bot]", "body": "No issues found", "updated_at": "2026-01-02T03:04:05Z"}}
	return signals
}

func withGates(signals map[string]any, gateID, status string) map[string]any {
	signals["gates"] = map[string]any{gateID: map[string]any{"status": status}}
	return signals
}

func workflowCaseStateIDs() []string {
	var out []string
	for _, tc := range goldenWorkflowCases() {
		out = append(out, tc.stateID)
	}
	sort.Strings(out)
	return out
}

func workflowCaseRouteIDs() []string {
	seen := map[string]bool{}
	for _, tc := range goldenWorkflowCases() {
		seen[tc.routeID] = true
	}
	return sortedKeys(seen)
}

func readWorkflowStateIDs(t *testing.T) []string {
	t.Helper()
	var doc struct {
		Rules []struct {
			StateID string `yaml:"state_id"`
		} `yaml:"rules"`
	}
	readYAML(t, ".reinguard/control/states/workflow.yaml", &doc)
	seen := map[string]bool{}
	for _, r := range doc.Rules {
		if r.StateID != "" {
			seen[r.StateID] = true
		}
	}
	return sortedKeys(seen)
}

func readWorkflowRouteIDs(t *testing.T) []string {
	t.Helper()
	var doc struct {
		Rules []struct {
			RouteID string `yaml:"route_id"`
		} `yaml:"rules"`
	}
	readYAML(t, ".reinguard/control/routes/workflow.yaml", &doc)
	seen := map[string]bool{}
	for _, r := range doc.Rules {
		if r.RouteID != "" {
			seen[r.RouteID] = true
		}
	}
	return sortedKeys(seen)
}

func readWorkflowGuardIDs(t *testing.T) []string {
	t.Helper()
	var doc struct {
		Rules []struct {
			GuardID string `yaml:"guard_id"`
		} `yaml:"rules"`
	}
	readYAML(t, ".reinguard/control/guards/default.yaml", &doc)
	seen := map[string]bool{}
	for _, r := range doc.Rules {
		if r.GuardID != "" {
			seen[r.GuardID] = true
		}
	}
	return sortedKeys(seen)
}

func readYAML(t *testing.T, rel string, out any) {
	t.Helper()
	if err := yaml.Unmarshal(readFile(t, filepath.Join(goldenRepoRoot(t), rel)), out); err != nil {
		t.Fatal(err)
	}
}

func sortedKeys(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

func containsString(xs []string, want string) bool {
	for _, x := range xs {
		if x == want {
			return true
		}
	}
	return false
}
