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
	"github.com/k-shibuki/reinguard/internal/resolve"
	"github.com/k-shibuki/reinguard/internal/schemaexport"
	"github.com/k-shibuki/reinguard/internal/signals"
	"github.com/k-shibuki/reinguard/pkg/schema"
	"github.com/urfave/cli/v2"
)

func isNonResolvedOutcome(k resolve.OutcomeKind) bool {
	return k == resolve.OutcomeAmbiguous || k == resolve.OutcomeDegraded || k == resolve.OutcomeUnsupported
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
		var ps []config.ProviderSpec
		for _, id := range providerOverride {
			ps = append(ps, config.ProviderSpec{ID: id, Enabled: true})
		}
		root = config.Root{SchemaVersion: loaded.Root.SchemaVersion, DefaultBranch: loaded.Root.DefaultBranch, Providers: ps}
	}
	branch, prNumber, err := parseObserveScopeFlags(c, false)
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
	}
	signals, diags, deg, err := engine.Collect(context.Background(), &root, opts)
	if err != nil {
		return err
	}
	doc := observation.Document(signals, diags, deg)
	return writeJSON(c.App.Writer, doc)
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
	if c.Bool("fail-on-non-resolved") && isNonResolvedOutcome(res.Kind) {
		return fmt.Errorf("non-resolved state outcome: %s", res.Kind)
	}
	return writeJSON(c.App.Writer, out)
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
	if c.Bool("fail-on-non-resolved") && isNonResolvedOutcome(res.Kind) {
		return fmt.Errorf("non-resolved route outcome: %s", res.Kind)
	}
	return writeJSON(c.App.Writer, out)
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
	useSig := c.String("observation-file") != ""
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
	sig, diags, deg, err := observe.LoadSignalsFileOrCollect(context.Background(), &loaded.Root, loadOpts)
	if err != nil {
		return err
	}
	obsDoc := observation.Document(sig, diags, deg)
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
	kEntries := make([]config.KnowledgeManifestEntry, 0)
	var kWarns []string
	if loaded.KnowledgePresent && loaded.Knowledge != nil {
		kEntries, kWarns = knowledge.FilterBySignals(loaded.Knowledge.Entries, flat)
		if kEntries == nil {
			kEntries = []config.KnowledgeManifestEntry{}
		}
	}
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
		"state":          stateRes,
		"routes":         []any{routeRes},
		"guards": map[string]any{
			"merge-readiness": gr,
		},
		"knowledge":   map[string]any{"entries": kEntries},
		"diagnostics": ctxDiags,
	}
	return writeJSON(c.App.Writer, ctxDoc)
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
	checks, err := loadChecksFile(wd, c.String("checks-file"))
	if err != nil {
		return err
	}
	art, err := gate.Record(context.Background(), cfgDir, wd, gateID, c.String("status"), checks, time.Time{})
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

func loadChecksFile(wd, path string) ([]gate.Check, error) {
	if path == "" {
		return []gate.Check{}, nil
	}
	data, err := os.ReadFile(resolveInputPath(wd, path))
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

func loadSignalsOpts(c *cli.Context, wd string) (observe.LoadSignalsOptions, error) {
	branch, prNumber, err := parseObserveScopeFlags(c, c.String("observation-file") != "")
	if err != nil {
		return observe.LoadSignalsOptions{}, err
	}
	opts := observe.LoadSignalsOptions{
		WorkDir:  wd,
		Branch:   branch,
		PRNumber: prNumber,
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
	for _, w := range config.DeprecatedWarnings(&res.Root) {
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
					Name:   "workflow-position",
					Usage:  "alias of full observe",
					Flags:  observeFlags(),
					Action: func(c *cli.Context) error { return RunObserve(c, "", nil) },
				},
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
						newGateChecksFileFlag(),
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
	hideHelpOnCommands(commands)

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
			newRootHelpFlag(),
			verFlag,
		},
		Commands: commands,
	}
}
