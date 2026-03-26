package rgdcli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/k-shibuki/reinguard/internal/config"
	"github.com/k-shibuki/reinguard/internal/configdir"
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

// RunObserve executes observation with optional provider / facet narrowing.
func RunObserve(c *cli.Context, gitHubFacet string, providerOverride []string) error {
	wd, cfgDir, err := resolvePaths(c)
	if err != nil {
		return err
	}
	loaded, err := config.Load(cfgDir)
	if err != nil {
		return err
	}
	engine := observe.NewEngine()
	opts := observe.Options{
		WorkDir:     wd,
		Serial:      c.Bool("serial"),
		ProviderIDs: providerOverride,
		GitHubFacet: gitHubFacet,
	}
	root := loaded.Root
	if len(providerOverride) > 0 {
		var ps []config.ProviderSpec
		for _, id := range providerOverride {
			ps = append(ps, config.ProviderSpec{ID: id, Enabled: true})
		}
		root = config.Root{SchemaVersion: loaded.Root.SchemaVersion, DefaultBranch: loaded.Root.DefaultBranch, Providers: ps}
	}
	signals, diags, deg, err := engine.Collect(context.Background(), &root, opts)
	if err != nil {
		return err
	}
	doc := observation.Document(signals, diags, deg)
	return writeJSON(c.App.Writer, doc)
}

// RunStateEval evaluates state rules.
func RunStateEval(c *cli.Context) error {
	wd, cfgDir, err := resolvePaths(c)
	if err != nil {
		return err
	}
	loaded, err := config.Load(cfgDir)
	if err != nil {
		return err
	}
	sig, diags, deg, err := observe.LoadSignalsFileOrCollect(context.Background(), &loaded.Root, loadSignalsOpts(c, wd))
	if err != nil {
		return err
	}
	degSet := observation.DegradedSet(diags, deg)
	res, err := resolve.ResolveState(loaded.Rules(), signals.Flatten(sig), degSet)
	if err != nil {
		return err
	}
	out := map[string]any{
		"kind":     string(res.Kind),
		"state_id": res.StateID,
		"route_id": res.RouteID,
		"rule_id":  res.RuleID,
		"priority": res.Priority,
		"reason":   res.Reason,
	}
	if len(res.Candidates) > 0 {
		out["candidates"] = res.Candidates
	}
	if c.Bool("fail-on-non-resolved") && (res.Kind == resolve.OutcomeAmbiguous || res.Kind == resolve.OutcomeDegraded) {
		return fmt.Errorf("non-resolved state outcome: %s", res.Kind)
	}
	return writeJSON(c.App.Writer, out)
}

// RunRouteSelect evaluates route rules.
func RunRouteSelect(c *cli.Context) error {
	wd, cfgDir, err := resolvePaths(c)
	if err != nil {
		return err
	}
	loaded, err := config.Load(cfgDir)
	if err != nil {
		return err
	}
	sig, diags, deg, err := observe.LoadSignalsFileOrCollect(context.Background(), &loaded.Root, loadSignalsOpts(c, wd))
	if err != nil {
		return err
	}
	flat := signals.Flatten(sig)
	if p := c.String("state-file"); p != "" {
		sf, rerr := os.ReadFile(resolveInputPath(wd, p))
		if rerr != nil {
			return rerr
		}
		var st map[string]any
		if jerr := json.Unmarshal(sf, &st); jerr != nil {
			return jerr
		}
		for k, v := range signals.Flatten(map[string]any{"state": st}) {
			flat[k] = v
		}
	}
	degSet := observation.DegradedSet(diags, deg)
	res, err := resolve.ResolveRoute(loaded.Rules(), flat, degSet)
	if err != nil {
		return err
	}
	out := map[string]any{
		"kind":     string(res.Kind),
		"state_id": res.StateID,
		"route_id": res.RouteID,
		"rule_id":  res.RuleID,
		"priority": res.Priority,
		"reason":   res.Reason,
	}
	if len(res.Candidates) > 0 {
		out["candidates"] = res.Candidates
	}
	if len(res.RouteCandidates) > 0 {
		rc := make([]any, len(res.RouteCandidates))
		for i, x := range res.RouteCandidates {
			rc[i] = map[string]any{"rule_id": x.RuleID, "route_id": x.RouteID, "priority": x.Priority}
		}
		out["route_candidates"] = rc
	}
	if c.Bool("fail-on-non-resolved") && (res.Kind == resolve.OutcomeAmbiguous || res.Kind == resolve.OutcomeDegraded) {
		return fmt.Errorf("non-resolved route outcome: %s", res.Kind)
	}
	return writeJSON(c.App.Writer, out)
}

// RunKnowledgePack lists knowledge entries from manifest (ADR-0010).
func RunKnowledgePack(c *cli.Context) error {
	_, cfgDir, err := resolvePaths(c)
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
	entries := knowledge.FilterByQuery(loaded.Knowledge.Entries, c.String("query"))
	return writeJSON(c.App.Writer, map[string]any{"entries": entries})
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
	sig, diags, deg, err := observe.LoadSignalsFileOrCollect(context.Background(), &loaded.Root, loadSignalsOpts(c, wd))
	if err != nil {
		return err
	}
	obsDoc := observation.Document(sig, diags, deg)
	flat := signals.Flatten(sig)
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
	routeRes, err := resolve.ResolveRoute(loaded.Rules(), flat, degSet)
	if err != nil {
		return err
	}
	gr := guard.EvalMergeReadiness(flat)
	kEntries := make([]config.KnowledgeManifestEntry, 0)
	if loaded.KnowledgePresent && loaded.Knowledge != nil {
		kEntries = append(kEntries, loaded.Knowledge.Entries...)
	}
	ctxDoc := map[string]any{
		"schema_version": schema.CurrentSchemaVersion,
		"observation":    obsDoc,
		"state":          stateRes,
		"routes":         []any{routeRes},
		"guards": map[string]any{
			"merge-readiness": gr,
		},
		"knowledge": map[string]any{"entries": kEntries},
		"diagnostics": append(diagsToMaps(diags), map[string]any{
			"severity": "info",
			"message":  "context build pipeline complete",
			"code":     "context_built",
		}),
	}
	return writeJSON(c.App.Writer, ctxDoc)
}

// RunGuardEval runs a named guard.
func RunGuardEval(c *cli.Context, guardID string) error {
	if guardID != "merge-readiness" {
		return fmt.Errorf("unknown guard %q", guardID)
	}
	path := c.String("observation-file")
	if path == "" {
		return fmt.Errorf("--observation-file is required")
	}
	wd, _, err := resolvePaths(c)
	if err != nil {
		return err
	}
	data, err := os.ReadFile(resolveInputPath(wd, path))
	if err != nil {
		return err
	}
	sig, _, _, err := observe.ParseObservationJSON(data)
	if err != nil {
		return err
	}
	res := guard.EvalMergeReadiness(sig)
	return writeJSON(c.App.Writer, res)
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

func loadSignalsOpts(c *cli.Context, wd string) observe.LoadSignalsOptions {
	opts := observe.LoadSignalsOptions{WorkDir: wd, Serial: c.Bool("serial")}
	if p := c.String("observation-file"); p != "" {
		opts.ObservationPath = resolveInputPath(wd, p)
	}
	return opts
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
					Name:   "pack",
					Usage:  "list knowledge entries from manifest",
					Flags:  []cli.Flag{newCwdFlag(), newConfigDirFlag(), newKnowledgeQueryFlag()},
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
					},
					Action: RunContextBuild,
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
