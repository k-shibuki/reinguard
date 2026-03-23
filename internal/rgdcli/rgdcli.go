package rgdcli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/k-shibuki/reinguard/internal/config"
	"github.com/k-shibuki/reinguard/internal/configdir"
	"github.com/k-shibuki/reinguard/internal/guard"
	"github.com/k-shibuki/reinguard/internal/observation"
	"github.com/k-shibuki/reinguard/internal/observe"
	"github.com/k-shibuki/reinguard/internal/resolve"
	"github.com/k-shibuki/reinguard/internal/schemaexport"
	"github.com/k-shibuki/reinguard/internal/validate"
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
	return writeJSON(c.App.Writer, doc, false)
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
	signals, diags, deg, err := loadOrObserve(c, wd, cfgDir, loaded)
	if err != nil {
		return err
	}
	degSet := observation.DegradedSet(diags, deg)
	res, err := resolve.ResolveState(loaded.Rules(), flattenSignals(signals), degSet)
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
	return writeJSON(c.App.Writer, out, false)
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
	signals, diags, deg, err := loadOrObserve(c, wd, cfgDir, loaded)
	if err != nil {
		return err
	}
	flat := flattenSignals(signals)
	if p := c.String("state-file"); p != "" {
		sf, rerr := os.ReadFile(p)
		if rerr != nil {
			return rerr
		}
		var st map[string]any
		if jerr := json.Unmarshal(sf, &st); jerr != nil {
			return jerr
		}
		flat["state"] = st
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
	return writeJSON(c.App.Writer, out, false)
}

// RunKnowledgePack lists knowledge paths from manifest.
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
		return writeJSON(c.App.Writer, map[string]any{"paths": []any{}}, false)
	}
	var paths []any
	for _, e := range loaded.Knowledge.Entries {
		paths = append(paths, e.Path)
	}
	return writeJSON(c.App.Writer, map[string]any{"paths": paths}, false)
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
	var signals map[string]any
	var diags []observe.Diagnostic
	var deg bool
	if p := c.String("observation-file"); p != "" {
		data, rerr := os.ReadFile(p)
		if rerr != nil {
			return rerr
		}
		signals, diags, deg, err = parseObservationJSON(data)
		if err != nil {
			return err
		}
	} else {
		engine := observe.NewEngine()
		signals, diags, deg, err = engine.Collect(context.Background(), &loaded.Root, observe.Options{WorkDir: wd, Serial: c.Bool("serial")})
		if err != nil {
			return err
		}
	}
	obsDoc := observation.Document(signals, diags, deg)
	flat := flattenSignals(signals)
	degSet := observation.DegradedSet(diags, deg)
	stateRes, err := resolve.ResolveState(loaded.Rules(), flat, degSet)
	if err != nil {
		return err
	}
	flat["state"] = map[string]any{
		"kind":     string(stateRes.Kind),
		"state_id": stateRes.StateID,
		"rule_id":  stateRes.RuleID,
	}
	routeRes, err := resolve.ResolveRoute(loaded.Rules(), flat, degSet)
	if err != nil {
		return err
	}
	gr := guard.EvalMergeReadiness(flat)
	var kpaths []any
	if loaded.KnowledgePresent && loaded.Knowledge != nil {
		for _, e := range loaded.Knowledge.Entries {
			kpaths = append(kpaths, e.Path)
		}
	}
	ctxDoc := map[string]any{
		"schema_version": schema.CurrentSchemaVersion,
		"observation":    obsDoc,
		"state":          stateRes,
		"routes":         []any{routeRes},
		"guards": map[string]any{
			"merge-readiness": gr,
		},
		"knowledge": map[string]any{"paths": kpaths},
		"diagnostics": append(diagsToMaps(diags), map[string]any{
			"severity": "info",
			"message":  "context build pipeline complete",
			"code":     "context_built",
		}),
	}
	return writeJSON(c.App.Writer, ctxDoc, false)
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
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return err
	}
	signals, _ := doc["signals"].(map[string]any)
	res := guard.EvalMergeReadiness(signals)
	return writeJSON(c.App.Writer, res, false)
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
	wd, err = configdir.WorkingDir()
	if err != nil {
		return "", "", err
	}
	if v := c.String("cwd"); v != "" {
		wd = v
	}
	cfgDir, err = configdir.Resolve(wd, c.String("config-dir"))
	return wd, cfgDir, err
}

func writeJSON(w io.Writer, v any, _ bool) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
}

func parseObservationJSON(data []byte) (signals map[string]any, diags []observe.Diagnostic, degraded bool, err error) {
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		return nil, nil, false, err
	}
	signals, _ = doc["signals"].(map[string]any)
	if signals == nil {
		signals = map[string]any{}
	}
	degraded, _ = doc["degraded"].(bool)
	if raw, ok := doc["diagnostics"].([]any); ok {
		for _, r := range raw {
			if m, ok := r.(map[string]any); ok {
				diags = append(diags, observe.Diagnostic{
					Severity: stringField(m, "severity"),
					Message:  stringField(m, "message"),
					Provider: stringField(m, "provider"),
					Code:     stringField(m, "code"),
				})
			}
		}
	}
	return signals, diags, degraded, nil
}

func loadOrObserve(c *cli.Context, wd, cfgDir string, loaded *config.LoadResult) (map[string]any, []observe.Diagnostic, bool, error) {
	if p := c.String("observation-file"); p != "" {
		data, err := os.ReadFile(p)
		if err != nil {
			return nil, nil, false, err
		}
		return parseObservationJSON(data)
	}
	engine := observe.NewEngine()
	signals, diags, deg, err := engine.Collect(context.Background(), &loaded.Root, observe.Options{WorkDir: wd, Serial: c.Bool("serial")})
	return signals, diags, deg, err
}

func flattenSignals(signals map[string]any) map[string]any {
	out := map[string]any{}
	var walk func(prefix string, v any)
	walk = func(prefix string, v any) {
		if m, ok := v.(map[string]any); ok {
			if prefix != "" {
				out[prefix] = v
			}
			for k, vv := range m {
				p := k
				if prefix != "" {
					p = prefix + "." + k
				}
				out[p] = vv
				walk(p, vv)
			}
			return
		}
		if prefix != "" {
			out[prefix] = v
		}
	}
	for ns, v := range signals {
		out[ns] = v
		walk(ns, v)
	}
	return out
}

func stringField(m map[string]any, k string) string {
	if v, ok := m[k].(string); ok {
		return v
	}
	return ""
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

func runConfigValidateLegacy(c *cli.Context) error {
	wd, err := configdir.WorkingDir()
	if err != nil {
		return err
	}
	if v := c.String("cwd"); v != "" {
		wd = v
	}
	dir, err := configdir.Resolve(wd, c.String("config-dir"))
	if err != nil {
		return err
	}
	if err = validate.Dir(dir); err != nil {
		return err
	}
	_, err = fmt.Fprintf(c.App.Writer, "config OK: %q\n", dir)
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
					Usage:  "validate .reinguard directory (legacy MVP check)",
					Flags:  []cli.Flag{newConfigDirFlag(), newCwdFlag()},
					Action: runConfigValidateLegacy,
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
					Name:   "pack",
					Usage:  "list knowledge paths from manifest",
					Flags:  []cli.Flag{newCwdFlag(), newConfigDirFlag()},
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
