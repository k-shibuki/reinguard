// Command rgd is the reinguard CLI (MVP).
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/k-shibuki/reinguard/internal/configdir"
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
						Name:  "validate",
						Usage: "validate .reinguard configuration (MVP stub)",
						Action: func(c *cli.Context) error {
							return runValidate(c)
						},
					},
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
