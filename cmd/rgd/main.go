// Command rgd is the reinguard CLI (MVP).
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/k-shibuki/reinguard/internal/configdir"
	"github.com/k-shibuki/reinguard/internal/labels"
	"github.com/k-shibuki/reinguard/internal/prbackfill"
	"github.com/k-shibuki/reinguard/internal/schemaexport"
	"github.com/k-shibuki/reinguard/internal/validate"
	"github.com/urfave/cli/v2"
)

// Set at link time: go build -ldflags "-X main.version=1.2.3".
var version = "dev"

func main() {
	app := &cli.App{
		Name:    "rgd",
		Usage:   "reinguard — spec-driven control-plane CLI",
		Version: version,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "config-dir",
				Usage:   "path to config directory (default: <git-root>/.reinguard)",
				EnvVars: []string{"REINGUARD_CONFIG_DIR"},
			},
		},
		Commands: []*cli.Command{
			{
				Name:  "version",
				Usage: "print rgd version",
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
						Usage:  "validate .reinguard configuration (MVP stub)",
						Action: runValidate,
					},
				},
			},
			{
				Name:  "ensure-labels",
				Usage: "create PR policy type and exception labels on the repo if missing (requires gh)",
				Description: "Idempotent maintainer command; replaces tools/gh-labels.sh. " +
					"Requires gh CLI with permission to manage labels.",
				Action: func(c *cli.Context) error {
					return labels.EnsureRepoLabels(c.App.Writer)
				},
			},
			{
				Name:  "backfill-pr-policy",
				Usage: "add missing PR template sections and type labels to open PRs (requires gh)",
				Description: "Uses gh api (not gh pr edit) because some gh versions exit non-zero on pr edit " +
					"when Classic Projects GraphQL is deprecated. Idempotent for sections already present.",
				Action: func(c *cli.Context) error {
					return prbackfill.Run(c.App.Writer)
				},
			},
			{
				Name:  "schema",
				Usage: "JSON Schema helpers",
				Subcommands: []*cli.Command{
					{
						Name:  "export",
						Usage: "write embedded placeholder schema(s) to a directory",
						Flags: []cli.Flag{
							&cli.StringFlag{
								Name:    "dir",
								Aliases: []string{"d"},
								Value:   "schema-export",
								Usage:   "output directory for schema files",
							},
						},
						Action: func(c *cli.Context) error {
							dir := c.String("dir")
							if err := schemaexport.Export(dir); err != nil {
								return err
							}
							_, err := fmt.Fprintf(c.App.Writer, "wrote embedded schema to %q\n", dir)
							return err
						},
					},
				},
			},
		},
		Action: func(c *cli.Context) error {
			if c.NArg() == 0 {
				return cli.ShowAppHelp(c)
			}
			return fmt.Errorf("unknown arguments: %v", c.Args().Slice())
		},
	}

	if err := app.Run(os.Args); err != nil {
		log.Fatal(err)
	}
}

func runValidate(c *cli.Context) error {
	cwd, err := configdir.WorkingDir()
	if err != nil {
		return err
	}
	cfgPath := c.String("config-dir")
	dir, err := configdir.Resolve(cwd, cfgPath)
	if err != nil {
		return err
	}
	if err = validate.Dir(dir); err != nil {
		return err
	}
	_, err = fmt.Fprintf(c.App.Writer, "config OK: %q\n", dir)
	return err
}
