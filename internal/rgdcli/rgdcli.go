// Package rgdcli implements rgd CLI command handlers by delegating to config load, observation,
// resolve, guard, knowledge, and schema export. User-facing command names, flags, and JSON
// output shapes are documented in docs/cli.md at the repository root.
//
// # ADR traceability
//
// ADR-0001 (Adapter / Semantics / Substrate; system vs runtime), ADR-0003 (observe and pull-based flow),
// ADR-0010 (knowledge pack and manifest), ADR-0011 (control plane and guards). Subcommands
// that touch observation and GitHub align with ADR-0005 and ADR-0006.
//
// # Errors
//
// Run* entry points return errors from path resolution, config load, I/O, observation,
// JSON decode, or resolve when the CLI is configured to fail on non-resolved outcomes.
package rgdcli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/k-shibuki/reinguard/internal/config"
	"github.com/k-shibuki/reinguard/internal/configdir"
	"github.com/k-shibuki/reinguard/internal/gate"
	"github.com/k-shibuki/reinguard/internal/guard"
	"github.com/k-shibuki/reinguard/internal/knowledge"
	"github.com/k-shibuki/reinguard/internal/observation"
	"github.com/k-shibuki/reinguard/internal/observe"
	"github.com/k-shibuki/reinguard/internal/procedure"
	"github.com/k-shibuki/reinguard/internal/resolve"
	"github.com/k-shibuki/reinguard/internal/schemaexport"
	"github.com/k-shibuki/reinguard/internal/signals"
	"github.com/k-shibuki/reinguard/pkg/schema"
	"github.com/urfave/cli/v2"
)

const exitCodeNonResolved = 2

type exitCodeError struct {
	code int
}

func (e *exitCodeError) Error() string {
	return "non-resolved outcome"
}

func (e *exitCodeError) ExitCode() int {
	return e.code
}

func isNonResolvedOutcome(k resolve.OutcomeKind) bool {
	return k == resolve.OutcomeAmbiguous || k == resolve.OutcomeDegraded || k == resolve.OutcomeUnsupported
}

func exitNonResolved() error {
	return &exitCodeError{code: exitCodeNonResolved}
}

func filterContextKnowledge(loaded *config.LoadResult, flat map[string]any) ([]config.KnowledgeManifestEntry, []string) {
	if !loaded.KnowledgePresent || loaded.Knowledge == nil {
		return []config.KnowledgeManifestEntry{}, nil
	}
	entries, warns := knowledge.FilterBySignals(loaded.Knowledge.Entries, flat)
	if entries == nil {
		entries = []config.KnowledgeManifestEntry{}
	}
	return entries, warns
}

func shouldFailContextBuild(strict bool, stateRes, routeRes resolve.Result) bool {
	return strict && (isNonResolvedOutcome(stateRes.Kind) || isNonResolvedOutcome(routeRes.Kind))
}

// resolveEvalOutputMap matches resolve.Result JSON omitempty so CLI stdout aligns with context build.
func resolveEvalOutputMap(res resolve.Result) map[string]any {
	out := map[string]any{
		"kind": string(res.Kind),
	}
	if res.StateID != "" {
		out["state_id"] = res.StateID
	}
	if res.RouteID != "" {
		out["route_id"] = res.RouteID
	}
	if res.GuardID != "" {
		out["guard_id"] = res.GuardID
	}
	if res.TargetID != "" {
		out["target_id"] = res.TargetID
	}
	if res.RuleID != "" {
		out["rule_id"] = res.RuleID
	}
	if res.Reason != "" {
		out["reason"] = res.Reason
	}
	if res.Kind == resolve.OutcomeResolved || res.Kind == resolve.OutcomeAmbiguous {
		out["priority"] = res.Priority
	}
	if len(res.Candidates) > 0 {
		out["candidates"] = res.Candidates
	}
	if len(res.MissingEvidence) > 0 {
		out["missing_evidence"] = res.MissingEvidence
	}
	if res.ReEntryHint != "" {
		out["re_entry_hint"] = res.ReEntryHint
	}
	return out
}

func contextBuildStateOutput(stateRes resolve.Result, routeRes resolve.Result, loaded *config.LoadResult) map[string]any {
	out := resolveEvalOutputMap(stateRes)
	if stateRes.Kind != resolve.OutcomeResolved || stateRes.StateID == "" || !loaded.ProcedurePresent {
		return out
	}
	routeOK := routeRes.Kind == resolve.OutcomeResolved && routeRes.RouteID != ""
	if e := procedure.HintEntry(loaded.ProcedureEntries, stateRes.StateID, routeOK, routeRes.RouteID); e != nil {
		out["procedure_hint"] = map[string]any{
			"procedure_id": e.ID,
			"path":         e.Path,
			"derived_from": "state_id",
		}
	}
	return out
}

// RunObserve loads config, runs the observation engine (see package observe), and writes
// observation JSON to the CLI writer. Provider overrides replace the configured provider list
// when non-empty. Errors propagate from config load, engine build, or Collect.
func RunObserve(c *cli.Context, gitHubFacet string, providerOverride []string) error {
	wd, cfgDir, err := resolvePaths(c)
	if err != nil {
		return err
	}
	loaded, err := config.Load(cfgDir)
	if err != nil {
		return err
	}
	root := loaded.Root
	if len(providerOverride) > 0 {
		ps, selectErr := selectProviderSpecs(loaded.Root.Providers, providerOverride)
		if selectErr != nil {
			return selectErr
		}
		root = config.Root{SchemaVersion: loaded.Root.SchemaVersion, DefaultBranch: loaded.Root.DefaultBranch, Providers: ps}
	}
	branch, prNumber, err := parseObserveScopeFlags(c, false)
	if err != nil {
		return err
	}
	view, err := parseObserveViewFlag(c, gitHubFacet, observe.ViewSummary)
	if err != nil {
		return err
	}
	engine, err := observe.NewEngineFromConfig(root.Providers)
	if err != nil {
		return err
	}
	opts := observe.Options{
		WorkDir:     wd,
		Serial:      c.Bool("serial"),
		ProviderIDs: providerOverride,
		GitHubFacet: gitHubFacet,
		Scope:       observe.Scope{Branch: branch, PRNumber: prNumber},
		View:        view,
	}
	signals, diags, deg, err := engine.Collect(context.Background(), &root, opts)
	if err != nil {
		return err
	}
	doc := observation.Document(signals, diags, deg, map[string]any{"view": string(view)})
	return writeJSON(c.App.Writer, doc)
}

func selectProviderSpecs(specs []config.ProviderSpec, providerIDs []string) ([]config.ProviderSpec, error) {
	selected := make([]config.ProviderSpec, 0, len(providerIDs))
	for _, wantID := range providerIDs {
		matched := false
		for _, spec := range specs {
			if spec.ID != wantID {
				continue
			}
			selected = append(selected, spec)
			matched = true
			break
		}
		if matched {
			continue
		}
		return nil, fmt.Errorf("provider override %q is not configured", wantID)
	}
	return selected, nil
}

// RunStateEval loads signals from --observation-file or live collect, then runs resolve on
// state rules. With --fail-on-non-resolved, ambiguous, degraded, or unsupported outcomes return an error.
func RunStateEval(c *cli.Context) error {
	wd, cfgDir, err := resolvePaths(c)
	if err != nil {
		return err
	}
	loaded, err := config.Load(cfgDir)
	if err != nil {
		return err
	}
	loadOpts, err := loadSignalsOpts(c, wd)
	if err != nil {
		return err
	}
	sig, diags, deg, err := observe.LoadSignalsFileOrCollect(context.Background(), &loaded.Root, loadOpts)
	if err != nil {
		return err
	}
	flat := signals.Flatten(sig)
	if mergeErr := mergeGateSignalsIntoFlat(context.Background(), cfgDir, wd, flat); mergeErr != nil {
		return mergeErr
	}
	degSet := observation.DegradedSet(diags, deg)
	res, err := resolve.ResolveState(loaded.Rules(), flat, degSet)
	if err != nil {
		return err
	}
	out := resolveEvalOutputMap(res)
	if err := writeJSON(c.App.Writer, out); err != nil {
		return err
	}
	if c.Bool("fail-on-non-resolved") && isNonResolvedOutcome(res.Kind) {
		return exitNonResolved()
	}
	return nil
}

// RunRouteSelect loads signals (and optional --state-file merge into the flat map), resolves
// route rules, and emits JSON. --fail-on-non-resolved turns ambiguous/degraded/unsupported into errors.
func RunRouteSelect(c *cli.Context) error {
	wd, cfgDir, err := resolvePaths(c)
	if err != nil {
		return err
	}
	loaded, err := config.Load(cfgDir)
	if err != nil {
		return err
	}
	loadOpts, err := loadSignalsOpts(c, wd)
	if err != nil {
		return err
	}
	sig, diags, deg, err := observe.LoadSignalsFileOrCollect(context.Background(), &loaded.Root, loadOpts)
	if err != nil {
		return err
	}
	flat := signals.Flatten(sig)
	if mergeErr := mergeGateSignalsIntoFlat(context.Background(), cfgDir, wd, flat); mergeErr != nil {
		return mergeErr
	}
	if mergeErr := mergeStateFileIntoFlat(wd, flat, c.String("state-file")); mergeErr != nil {
		return mergeErr
	}
	degSet := observation.DegradedSet(diags, deg)
	res, err := resolve.ResolveRoute(loaded.Rules(), flat, degSet)
	if err != nil {
		return err
	}
	out := resolveEvalOutputMap(res)
	if len(res.RouteCandidates) > 0 {
		rc := make([]any, len(res.RouteCandidates))
		for i, x := range res.RouteCandidates {
			rc[i] = map[string]any{"rule_id": x.RuleID, "route_id": x.RouteID, "priority": x.Priority}
		}
		out["route_candidates"] = rc
	}
	if err := writeJSON(c.App.Writer, out); err != nil {
		return err
	}
	if c.Bool("fail-on-non-resolved") && isNonResolvedOutcome(res.Kind) {
		return exitNonResolved()
	}
	return nil
}

// RunKnowledgePack lists knowledge entries from manifest (ADR-0010).
func RunKnowledgePack(c *cli.Context) error {
	wd, cfgDir, err := resolvePaths(c)
	if err != nil {
		return err
	}
	loaded, err := config.Load(cfgDir)
	if err != nil {
		return err
	}
	if !loaded.KnowledgePresent || loaded.Knowledge == nil {
		return writeJSON(c.App.Writer, map[string]any{"entries": []config.KnowledgeManifestEntry{}})
	}
	useSig := c.String("observation-file") != "" || c.IsSet("branch") || c.IsSet("pr")
	var flat map[string]any
	if useSig {
		loadOpts, lerr := loadSignalsOpts(c, wd)
		if lerr != nil {
			return lerr
		}
		sig, _, _, lerr := observe.LoadSignalsFileOrCollect(context.Background(), &loaded.Root, loadOpts)
		if lerr != nil {
			return lerr
		}
		flat = signals.Flatten(sig)
	}
	entries, warns := knowledge.FilterUnion(loaded.Knowledge.Entries, flat, useSig, c.String("query"))
	out := map[string]any{"entries": entries}
	if len(warns) > 0 {
		var diags []any
		for _, w := range warns {
			diags = append(diags, map[string]any{
				"severity": "warning",
				"message":  w,
				"code":     "knowledge_when_eval",
			})
		}
		out["diagnostics"] = diags
	}
	return writeJSON(c.App.Writer, out)
}

// RunContextBuild runs the default operational-context pipeline.
func RunContextBuild(c *cli.Context) error {
	wd, cfgDir, err := resolvePaths(c)
	if err != nil {
		return err
	}
	loaded, err := config.Load(cfgDir)
	if err != nil {
		return err
	}
	loadOpts, err := loadSignalsOpts(c, wd)
	if err != nil {
		return err
	}
	sig, diags, deg, obsView, err := loadContextObservation(context.Background(), &loaded.Root, loadOpts)
	if err != nil {
		return err
	}
	obsDoc := observation.Document(sig, diags, deg, map[string]any{"view": string(obsView)})
	flat := signals.Flatten(sig)
	if mergeErr := mergeGateSignalsIntoFlat(context.Background(), cfgDir, wd, flat); mergeErr != nil {
		return mergeErr
	}
	degSet := observation.DegradedSet(diags, deg)
	stateRes, err := resolve.ResolveState(loaded.Rules(), flat, degSet)
	if err != nil {
		return err
	}
	for k, v := range signals.Flatten(map[string]any{
		"state": map[string]any{
			"kind":     string(stateRes.Kind),
			"state_id": stateRes.StateID,
			"rule_id":  stateRes.RuleID,
		},
	}) {
		flat[k] = v
	}
	kEntries, kWarns := filterContextKnowledge(loaded, flat)
	routeRes, err := resolve.ResolveRoute(loaded.Rules(), flat, degSet)
	if err != nil {
		return err
	}
	gr := guard.EvalWithRules(loaded.Rules(), guard.DefaultRegistry(), "merge-readiness", flat, degSet)
	ctxDiags := diagsToMaps(diags)
	for _, w := range kWarns {
		ctxDiags = append(ctxDiags, map[string]any{
			"severity": "warning",
			"message":  w,
			"code":     "knowledge_when_eval",
		})
	}
	ctxDiags = append(ctxDiags, map[string]any{
		"severity": "info",
		"message":  "context build pipeline complete",
		"code":     "context_built",
	})
	ctxDoc := map[string]any{
		"schema_version": schema.CurrentSchemaVersion,
		"observation":    obsDoc,
		"state":          contextBuildStateOutput(stateRes, routeRes, loaded),
		"routes":         []any{routeRes},
		"guards": map[string]any{
			"merge-readiness": gr,
		},
		"knowledge":   map[string]any{"entries": kEntries},
		"diagnostics": ctxDiags,
	}
	if c.Bool("compact") {
		ctxDoc["observation"] = compactObservationDocument(obsDoc)
	}
	if err := writeJSON(c.App.Writer, ctxDoc); err != nil {
		return err
	}
	if shouldFailContextBuild(c.Bool("fail-on-non-resolved"), stateRes, routeRes) {
		return exitNonResolved()
	}
	return nil
}

func loadContextObservation(ctx context.Context, root *config.Root, opts observe.LoadSignalsOptions) (map[string]any, []observe.Diagnostic, bool, observe.View, error) {
	if opts.ObservationPath == "" {
		sig, diags, deg, err := observe.LoadSignalsFileOrCollect(ctx, root, opts)
		return sig, diags, deg, opts.View, err
	}
	doc, err := observe.LoadObservationFile(opts.ObservationPath)
	if err != nil {
		return nil, nil, false, "", err
	}
	view := opts.View
	if raw, ok := doc.Meta["view"].(string); ok {
		if detected := observe.View(strings.TrimSpace(raw)); detected.Valid() {
			view = detected
		} else {
			doc.Diagnostics = append(doc.Diagnostics, observe.Diagnostic{
				Severity: "warning",
				Message:  fmt.Sprintf("observation file contains invalid view %q, using default", raw),
				Code:     "invalid_observation_view",
			})
		}
	}
	return doc.Signals, doc.Diagnostics, doc.Degraded, view, nil
}

// RunGuardEval runs a named guard.
func RunGuardEval(c *cli.Context, guardID string) error {
	path := c.String("observation-file")
	if path == "" {
		return fmt.Errorf("--observation-file is required")
	}
	wd, cfgDir, err := resolvePaths(c)
	if err != nil {
		return err
	}
	if _, ok := guard.DefaultRegistry().Lookup(guardID); !ok {
		return fmt.Errorf("unknown guard %q", guardID)
	}
	loaded, err := config.Load(cfgDir)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(resolveInputPath(wd, path))
	if err != nil {
		return err
	}
	sig, diags, deg, err := observe.ParseObservationJSON(data)
	if err != nil {
		return err
	}
	flat := signals.Flatten(sig)
	if err := mergeGateSignalsIntoFlat(context.Background(), cfgDir, wd, flat); err != nil {
		return err
	}
	degSet := observation.DegradedSet(diags, deg)
	res := guard.EvalWithRules(loaded.Rules(), guard.DefaultRegistry(), guardID, flat, degSet)
	return writeJSON(c.App.Writer, res)
}

// RunGateRecord records one runtime gate artifact for the current branch HEAD.
func RunGateRecord(c *cli.Context, gateID string) error {
	wd, cfgDir, err := resolvePaths(c)
	if err != nil {
		return err
	}
	checks, err := loadChecksFile(c.App.Reader, wd, c.String("checks-file"))
	if err != nil {
		return err
	}
	jsonChecks, err := parseInlineCheckJSON(c.String("check-json"))
	if err != nil {
		return err
	}
	inlineChecks, err := parseInlineChecks(c.StringSlice("check"))
	if err != nil {
		return err
	}
	checks = append(checks, jsonChecks...)
	checks = append(checks, inlineChecks...)
	if len(checks) == 0 {
		return fmt.Errorf("gate record: at least one check entry is required (provide --check, --check-json, or --checks-file)")
	}
	inputs, err := loadGateInputsFile(wd, c.String("inputs-file"))
	if err != nil {
		return err
	}
	inputGateIDs := c.StringSlice("input-gate")
	resolvedInputs, err := resolveInputGates(context.Background(), cfgDir, wd, inputGateIDs)
	if err != nil {
		return err
	}
	inputs = append(inputs, resolvedInputs...)
	producer := gate.Producer{
		Procedure: c.String("producer-procedure"),
		Tool:      c.String("producer-tool"),
	}
	art, err := gate.Record(context.Background(), cfgDir, wd, gateID, c.String("status"), producer, inputs, checks, time.Time{})
	if err != nil {
		return err
	}
	return writeJSON(c.App.Writer, art)
}

// RunGateStatus prints the derived gate status for the current branch HEAD.
func RunGateStatus(c *cli.Context, gateID string) error {
	wd, cfgDir, err := resolvePaths(c)
	if err != nil {
		return err
	}
	res, err := gate.Status(context.Background(), cfgDir, wd, gateID)
	if err != nil {
		return err
	}
	return writeJSON(c.App.Writer, res)
}

// RunGateShow prints the validated stored gate artifact.
func RunGateShow(c *cli.Context, gateID string) error {
	_, cfgDir, err := resolvePaths(c)
	if err != nil {
		return err
	}
	art, err := gate.Show(cfgDir, gateID)
	if err != nil {
		return err
	}
	return writeJSON(c.App.Writer, art)
}

// RunSchemaExport exports embedded schemas.
func RunSchemaExport(c *cli.Context) error {
	dir := c.String("dir")
	if err := schemaexport.Export(dir); err != nil {
		return err
	}
	_, err := fmt.Fprintf(c.App.Writer, "exported schemas to %q\n", dir)
	return err
}

func resolvePaths(c *cli.Context) (wd string, cfgDir string, err error) {
	if v := c.String("cwd"); v != "" {
		wd = v
	} else {
		wd, err = configdir.WorkingDir()
		if err != nil {
			return "", "", err
		}
	}
	cfgDir, err = configdir.Resolve(wd, c.String("config-dir"))
	return wd, cfgDir, err
}

// resolveInputPath joins a user path against baseDir when the path is relative.
func resolveInputPath(baseDir, p string) string {
	if p == "" {
		return p
	}
	if filepath.IsAbs(p) {
		return p
	}
	return filepath.Join(baseDir, p)
}

func mergeStateFileIntoFlat(wd string, flat map[string]any, path string) error {
	if path == "" {
		return nil
	}
	sf, err := os.ReadFile(resolveInputPath(wd, path))
	if err != nil {
		return err
	}
	var st map[string]any
	if err := json.Unmarshal(sf, &st); err != nil {
		return err
	}
	for k, v := range signals.Flatten(map[string]any{"state": st}) {
		flat[k] = v
	}
	return nil
}

func mergeGateSignalsIntoFlat(ctx context.Context, cfgDir, wd string, flat map[string]any) error {
	gs, err := gate.LoadSignals(ctx, cfgDir, wd)
	if err != nil {
		return err
	}
	for k, v := range signals.Flatten(gs) {
		flat[k] = v
	}
	return nil
}

func parseInlineChecks(raw []string) ([]gate.Check, error) {
	var out []gate.Check
	for _, s := range raw {
		parts := strings.SplitN(s, ":", 3)
		if len(parts) < 3 || parts[0] == "" || parts[1] == "" || parts[2] == "" {
			return nil, fmt.Errorf("gate record: --check value %q must be id:status:summary", s)
		}
		out = append(out, gate.Check{
			ID:      parts[0],
			Status:  parts[1],
			Summary: parts[2],
		})
	}
	return out, nil
}

func parseInlineCheckJSON(raw string) ([]gate.Check, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []gate.Check{}, nil
	}
	var many []gate.Check
	if err := json.Unmarshal([]byte(raw), &many); err == nil {
		if many == nil {
			return []gate.Check{}, nil
		}
		return many, nil
	}
	var one gate.Check
	if err := json.Unmarshal([]byte(raw), &one); err != nil {
		return nil, fmt.Errorf("gate record: --check-json must decode to one check object or an array of check objects: %w", err)
	}
	return []gate.Check{one}, nil
}

func loadChecksFile(stdin io.Reader, wd, path string) ([]gate.Check, error) {
	if path == "" {
		return []gate.Check{}, nil
	}
	var (
		data []byte
		err  error
	)
	if path == "-" {
		if stdin == nil {
			return nil, fmt.Errorf("gate record: --checks-file - requires stdin")
		}
		data, err = io.ReadAll(stdin)
	} else {
		data, err = os.ReadFile(resolveInputPath(wd, path))
	}
	if err != nil {
		return nil, err
	}
	var checks []gate.Check
	if err := json.Unmarshal(data, &checks); err != nil {
		return nil, err
	}
	if checks == nil {
		return []gate.Check{}, nil
	}
	return checks, nil
}

func loadGateInputsFile(wd, path string) ([]gate.Input, error) {
	if path == "" {
		return []gate.Input{}, nil
	}
	data, err := os.ReadFile(resolveInputPath(wd, path))
	if err != nil {
		return nil, err
	}
	var inputs []gate.Input
	if err := json.Unmarshal(data, &inputs); err != nil {
		return nil, err
	}
	if inputs == nil {
		return []gate.Input{}, nil
	}
	return inputs, nil
}

func resolveInputGates(ctx context.Context, cfgDir, wd string, gateIDs []string) ([]gate.Input, error) {
	if len(gateIDs) == 0 {
		return []gate.Input{}, nil
	}
	out := make([]gate.Input, 0, len(gateIDs))
	seen := map[string]struct{}{}
	for _, gateID := range gateIDs {
		gateID = strings.TrimSpace(gateID)
		if gateID == "" {
			return nil, fmt.Errorf("--input-gate must be non-empty")
		}
		if _, ok := seen[gateID]; ok {
			return nil, fmt.Errorf("duplicate --input-gate %q", gateID)
		}
		seen[gateID] = struct{}{}
		res, err := gate.Status(ctx, cfgDir, wd, gateID)
		if err != nil {
			return nil, err
		}
		if res.Status != gate.StatusPass {
			return nil, fmt.Errorf("--input-gate %q must be fresh pass, got %q", gateID, res.Status)
		}
		art, err := gate.Show(cfgDir, gateID)
		if err != nil {
			return nil, err
		}
		out = append(out, gate.Input{
			GateID:     art.GateID,
			Status:     art.Status,
			Subject:    art.Subject,
			RecordedAt: art.RecordedAt,
		})
	}
	return out, nil
}

func loadSignalsOpts(c *cli.Context, wd string) (observe.LoadSignalsOptions, error) {
	branch, prNumber, err := parseObserveScopeFlags(c, c.String("observation-file") != "")
	if err != nil {
		return observe.LoadSignalsOptions{}, err
	}
	opts := observe.LoadSignalsOptions{
		WorkDir:  wd,
		Branch:   branch,
		PRNumber: prNumber,
		View:     observe.ViewFull,
		Serial:   c.Bool("serial"),
	}
	if p := c.String("observation-file"); p != "" {
		opts.ObservationPath = resolveInputPath(wd, p)
	}
	return opts, nil
}

func parseObserveScopeFlags(c *cli.Context, observationFileSet bool) (branch string, prNumber int, err error) {
	branch = strings.TrimSpace(c.String("branch"))
	if c.IsSet("branch") && branch == "" {
		return "", 0, fmt.Errorf("--branch must be non-empty")
	}
	if c.IsSet("pr") {
		prNumber = c.Int("pr")
		if prNumber <= 0 {
			return "", 0, fmt.Errorf("--pr must be greater than 0")
		}
	}
	if observationFileSet && (branch != "" || prNumber > 0) {
		return "", 0, fmt.Errorf("--branch/--pr cannot be used with --observation-file")
	}
	return branch, prNumber, nil
}

func parseObserveViewFlag(c *cli.Context, gitHubFacet string, defaultView observe.View) (observe.View, error) {
	raw := strings.TrimSpace(c.String("view"))
	if raw == "" {
		return defaultView, nil
	}
	view := observe.View(raw)
	if !view.Valid() {
		return "", fmt.Errorf("--view must be one of %s", strings.Join(observeViewChoices(), ", "))
	}
	if view == observe.ViewInbox && gitHubFacet != "reviews" {
		return "", fmt.Errorf("--view inbox is only supported for rgd observe github reviews")
	}
	return view, nil
}

func observeViewChoices() []string {
	return []string{
		string(observe.ViewSummary),
		string(observe.ViewInbox),
		string(observe.ViewFull),
	}
}

func compactObservationDocument(doc map[string]any) map[string]any {
	out := map[string]any{
		"schema_version": doc["schema_version"],
		"degraded":       doc["degraded"],
	}
	if diags, ok := doc["diagnostics"]; ok {
		out["diagnostics"] = diags
	}
	if meta, ok := doc["meta"]; ok {
		out["meta"] = meta
	}
	signalsAny, ok := doc["signals"].(map[string]any)
	if !ok {
		out["signals"] = map[string]any{}
		return out
	}
	out["signals"] = compactObservationSignals(signalsAny)
	return out
}

func compactObservationSignals(signalsAny map[string]any) map[string]any {
	out := map[string]any{}
	if gitAny, ok := signalsAny["git"]; ok {
		out["git"] = gitAny
	}
	ghAny, ok := signalsAny["github"].(map[string]any)
	if !ok {
		return out
	}
	ghOut := map[string]any{}
	if repo := compactSelectedKeys(ghAny, "repository", "owner", "name", "identity_source"); len(repo) > 0 {
		ghOut["repository"] = repo
	}
	if issues := compactSelectedKeys(ghAny, "issues", "open_count"); len(issues) > 0 {
		ghOut["issues"] = issues
	}
	if prs := compactSelectedKeys(ghAny, "pull_requests",
		"open_count", "current_branch", "pr_exists_for_branch", "pr_number_for_branch",
		"state", "draft", "title", "base_ref", "head_ref", "head_sha", "mergeable",
		"merge_state_status", "observed_scope",
	); len(prs) > 0 {
		ghOut["pull_requests"] = prs
	}
	if ci := compactSelectedKeys(ghAny, "ci", "ci_status", "head_sha"); len(ci) > 0 {
		ghOut["ci"] = ci
	}
	if reviews := compactSelectedKeys(ghAny, "reviews",
		"review_threads_total", "review_threads_unresolved", "pagination_incomplete",
		"review_decisions_total", "review_decisions_approved", "review_decisions_changes_requested",
		"review_decisions_truncated", "bot_review_diagnostics",
		// bot_reviewer_status is preserved in compact mode because wait-bot-review.md
		// (.reinguard/procedure/wait-bot-review.md) instructs agents to read it from
		// observation.signals.github.reviews.bot_reviewer_status — including out of
		// --compact payloads — to explain waiting_bot_rate_limited / waiting_bot_paused.
		"bot_reviewer_status",
	); len(reviews) > 0 {
		ghOut["reviews"] = reviews
	}
	if len(ghOut) > 0 {
		out["github"] = ghOut
	}
	return out
}

// compactSelectedKeys keeps only the fields that remain useful in compact mode:
// workflow-driving scalars, scope metadata, and merge/readiness aggregates.
func compactSelectedKeys(parent map[string]any, field string, keys ...string) map[string]any {
	raw, ok := parent[field].(map[string]any)
	if !ok {
		return nil
	}
	out := map[string]any{}
	for _, key := range keys {
		if v, exists := raw[key]; exists {
			out[key] = v
		}
	}
	return out
}

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func diagsToMaps(diags []observe.Diagnostic) []any {
	var out []any
	for _, d := range diags {
		out = append(out, map[string]any{
			"severity": d.Severity,
			"message":  d.Message,
			"provider": d.Provider,
			"code":     d.Code,
		})
	}
	return out
}

func runConfigValidate(c *cli.Context) error {
	var wd string
	var err error
	if v := c.String("cwd"); v != "" {
		wd = v
	} else {
		wd, err = configdir.WorkingDir()
		if err != nil {
			return err
		}
	}
	dir, err := configdir.Resolve(wd, c.String("config-dir"))
	if err != nil {
		return err
	}
	res, err := config.Load(dir)
	if err != nil {
		return err
	}
	if _, perr := observe.NewEngineFromConfig(res.Root.Providers); perr != nil {
		return fmt.Errorf("config: provider build: %w", perr)
	}
	for _, w := range config.ConfigWarnings(res) {
		_, _ = fmt.Fprintln(c.App.ErrWriter, w)
	}
	repoRoot := configdir.RepoRoot(dir)
	knowledgeDir := filepath.Join(dir, "knowledge")
	if res.KnowledgePresent && res.Knowledge != nil {
		if e := knowledge.ValidateEntryPaths(repoRoot, res.Knowledge); e != nil {
			return e
		}
		if e := knowledge.CheckFreshness(res.Knowledge, repoRoot, knowledgeDir); e != nil {
			return e
		}
		for _, w := range knowledge.HintWarnings(repoRoot, res.Knowledge) {
			_, _ = fmt.Fprintln(c.App.ErrWriter, w)
		}
	}
	_, err = fmt.Fprintf(c.App.Writer, "config OK: %q\n", dir)
	return err
}

func runKnowledgeIndex(c *cli.Context) error {
	_, cfgDir, err := resolvePaths(c)
	if err != nil {
		return err
	}
	repoRoot := configdir.RepoRoot(cfgDir)
	knowledgeDir := filepath.Join(cfgDir, "knowledge")
	m, err := knowledge.BuildManifest(repoRoot, knowledgeDir)
	if err != nil {
		return err
	}
	outPath := filepath.Join(knowledgeDir, "manifest.json")
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	data = append(data, '\n')
	if werr := os.WriteFile(outPath, data, 0o644); werr != nil {
		return werr
	}
	_, err = fmt.Fprintf(c.App.Writer, "wrote %d entries to %s\n", len(m.Entries), outPath)
	return err
}

// NewApp builds the urfave CLI application (observe + state eval; extended in later PRs).
func NewApp(version string) *cli.App {
	verFlag := newRootVersionFlag()
	verFlag.Action = func(c *cli.Context, printVer bool) error {
		if printVer {
			cli.ShowVersion(c)
			return cli.Exit("", 0)
		}
		return nil
	}

	commands := []*cli.Command{
		{
			Name:  "version",
			Usage: "print version",
			Action: func(c *cli.Context) error {
				_, err := fmt.Fprintln(c.App.Writer, c.App.Version)
				return err
			},
		},
		{
			Name:  "config",
			Usage: "configuration commands",
			Subcommands: []*cli.Command{
				{
					Name:   "validate",
					Usage:  "validate .reinguard against JSON Schemas",
					Flags:  []cli.Flag{newConfigDirFlag(), newCwdFlag()},
					Action: runConfigValidate,
				},
			},
		},
		{
			Name:  "schema",
			Usage: "JSON Schema helpers",
			Subcommands: []*cli.Command{
				{
					Name:   "export",
					Usage:  "write embedded schemas to a directory",
					Flags:  []cli.Flag{newSchemaExportDirFlag()},
					Action: RunSchemaExport,
				},
			},
		},
		{
			Name:   "observe",
			Usage:  "collect observation JSON",
			Flags:  observeFlags(),
			Action: func(c *cli.Context) error { return RunObserve(c, "", nil) },
			Subcommands: []*cli.Command{
				{
					Name:   "git",
					Usage:  "git provider only",
					Flags:  observeFlags(),
					Action: func(c *cli.Context) error { return RunObserve(c, "", []string{"git"}) },
				},
				{
					Name:   "github",
					Usage:  "all GitHub facets",
					Flags:  observeFlags(),
					Action: func(c *cli.Context) error { return RunObserve(c, "", []string{"github"}) },
					Subcommands: []*cli.Command{
						{
							Name:   "issues",
							Flags:  observeFlags(),
							Action: func(c *cli.Context) error { return RunObserve(c, "issues", []string{"github"}) },
						},
						{
							Name:  "pull-requests",
							Flags: observeFlags(),
							Action: func(c *cli.Context) error {
								return RunObserve(c, "pull-requests", []string{"github"})
							},
						},
						{
							Name:   "ci",
							Flags:  observeFlags(),
							Action: func(c *cli.Context) error { return RunObserve(c, "ci", []string{"github"}) },
						},
						{
							Name:   "reviews",
							Flags:  observeFlags(),
							Action: func(c *cli.Context) error { return RunObserve(c, "reviews", []string{"github"}) },
						},
					},
				},
			},
		},
		{
			Name:  "state",
			Usage: "state evaluation",
			Subcommands: []*cli.Command{
				{
					Name:  "eval",
					Usage: "evaluate state rules",
					Flags: []cli.Flag{
						newObservationFileFlag(),
						newBranchFlag(),
						newPRNumberFlag(),
						newSerialFlag(),
						newCwdFlag(),
						newConfigDirFlag(),
						newFailOnNonResolvedFlag(),
					},
					Action: RunStateEval,
				},
			},
		},
		{
			Name:  "route",
			Usage: "route selection",
			Subcommands: []*cli.Command{
				{
					Name:  "select",
					Usage: "evaluate route rules",
					Flags: []cli.Flag{
						newObservationFileFlag(),
						newStateFileFlag(),
						newBranchFlag(),
						newPRNumberFlag(),
						newSerialFlag(),
						newCwdFlag(),
						newConfigDirFlag(),
						newFailOnNonResolvedFlag(),
					},
					Action: RunRouteSelect,
				},
			},
		},
		{
			Name:  "knowledge",
			Usage: "knowledge commands",
			Subcommands: []*cli.Command{
				{
					Name:   "index",
					Usage:  "generate knowledge/manifest.json from Markdown front matter",
					Flags:  []cli.Flag{newCwdFlag(), newConfigDirFlag()},
					Action: runKnowledgeIndex,
				},
				{
					Name:  "pack",
					Usage: "list knowledge entries from manifest",
					Flags: []cli.Flag{
						newCwdFlag(),
						newConfigDirFlag(),
						newKnowledgeQueryFlag(),
						newObservationFileFlag(),
						newBranchFlag(),
						newPRNumberFlag(),
					},
					Action: RunKnowledgePack,
				},
			},
		},
		{
			Name:  "context",
			Usage: "operational context",
			Subcommands: []*cli.Command{
				{
					Name:  "build",
					Usage: "full pipeline",
					Flags: []cli.Flag{
						newSerialFlag(),
						newCwdFlag(),
						newConfigDirFlag(),
						newObservationFileFlag(),
						newBranchFlag(),
						newPRNumberFlag(),
						newCompactFlag(),
						newFailOnNonResolvedFlag(),
					},
					Action: RunContextBuild,
				},
			},
		},
		{
			Name:  "gate",
			Usage: "runtime gate artifacts",
			Subcommands: []*cli.Command{
				{
					Name:      "record",
					Usage:     "record a gate artifact for the current HEAD",
					ArgsUsage: "<gate-id>",
					Flags: []cli.Flag{
						newGateStatusFlag(),
						newGateCheckFlag(),
						newGateCheckJSONFlag(),
						newGateChecksFileFlag(),
						newGateInputsFileFlag(),
						newGateInputGateFlag(),
						newGateProducerProcedureFlag(),
						newGateProducerToolFlag(),
						newCwdFlag(),
						newConfigDirFlag(),
					},
					Action: func(c *cli.Context) error {
						if c.NArg() < 1 {
							return fmt.Errorf("gate id required")
						}
						return RunGateRecord(c, c.Args().First())
					},
				},
				{
					Name:      "status",
					Usage:     "derive gate freshness and status for the current HEAD",
					ArgsUsage: "<gate-id>",
					Flags: []cli.Flag{
						newCwdFlag(),
						newConfigDirFlag(),
					},
					Action: func(c *cli.Context) error {
						if c.NArg() < 1 {
							return fmt.Errorf("gate id required")
						}
						return RunGateStatus(c, c.Args().First())
					},
				},
				{
					Name:      "show",
					Usage:     "print a stored gate artifact",
					ArgsUsage: "<gate-id>",
					Flags: []cli.Flag{
						newCwdFlag(),
						newConfigDirFlag(),
					},
					Action: func(c *cli.Context) error {
						if c.NArg() < 1 {
							return fmt.Errorf("gate id required")
						}
						return RunGateShow(c, c.Args().First())
					},
				},
			},
		},
		{
			Name:  "guard",
			Usage: "guard evaluation",
			Subcommands: []*cli.Command{
				{
					Name:      "eval",
					Usage:     "evaluate a guard by id",
					ArgsUsage: "<guard-id>",
					Flags: []cli.Flag{
						newObservationFileRequiredFlag(),
						newCwdFlag(),
						newConfigDirFlag(),
					},
					Action: func(c *cli.Context) error {
						if c.NArg() < 1 {
							return fmt.Errorf("guard id required")
						}
						return RunGuardEval(c, c.Args().First())
					},
				},
			},
		},
		{
			Name:  "review",
			Usage: "pull request review transport helpers",
			Subcommands: []*cli.Command{
				{
					Name:   "reply-thread",
					Usage:  "post a threaded pull-request review reply",
					Flags:  reviewReplyThreadFlags(),
					Action: runReviewReplyThread,
				},
				{
					Name:   "resolve-thread",
					Usage:  "resolve a pull-request review thread",
					Flags:  reviewResolveThreadFlags(),
					Action: runReviewResolveThread,
				},
			},
		},
		{
			Name:   "ensure-labels",
			Usage:  "create missing GitHub labels from .reinguard/labels.yaml",
			Flags:  []cli.Flag{newCwdFlag(), newConfigDirFlag()},
			Action: runEnsureLabels,
		},
		{
			Name:  "labels",
			Usage: "GitHub label helpers (repository tooling)",
			Subcommands: []*cli.Command{
				{
					Name:   "list",
					Usage:  "print label names as JSON (stdout)",
					Flags:  []cli.Flag{newCwdFlag(), newConfigDirFlag(), newLabelsCategoryFlag()},
					Action: runLabelsList,
				},
				{
					Name:   "sync",
					Usage:  "sync GitHub label color and description from labels.yaml",
					Flags:  []cli.Flag{newCwdFlag(), newConfigDirFlag(), newDryRunFlag()},
					Action: runLabelsSync,
				},
			},
		},
	}
	addHelpFlagOnCommands(commands)

	return &cli.App{
		Name:        "rgd",
		Version:     version,
		Usage:       "reinguard — spec-driven control-plane CLI",
		HideHelp:    true,
		HideVersion: true,
		Flags: []cli.Flag{
			newConfigDirFlag(),
			newCwdFlag(),
			newSerialFlag(),
			newFailOnNonResolvedFlag(),
			newHelpFlag(),
			verFlag,
		},
		Commands: commands,
	}
}
