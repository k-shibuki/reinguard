package rgdcli

import (
	"context"
	"encoding/json"
	"fmt"
	"io"

	"github.com/k-shibuki/reinguard/internal/config"
	"github.com/k-shibuki/reinguard/internal/configdir"
	"github.com/k-shibuki/reinguard/internal/observation"
	"github.com/k-shibuki/reinguard/internal/observe"
	"github.com/k-shibuki/reinguard/internal/schemaexport"
	"github.com/k-shibuki/reinguard/internal/validate"
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

func writeJSON(w io.Writer, v any) error {
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(v)
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

// NewApp builds the urfave CLI application (observe surface; extended in later PRs).
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
