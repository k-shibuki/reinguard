package rgdcli

import (
	"encoding/json"
	"fmt"
	"sort"

	"github.com/k-shibuki/reinguard/internal/labels"
	"github.com/urfave/cli/v2"
)

func runEnsureLabels(c *cli.Context) error {
	_, cfgDir, err := resolvePaths(c)
	if err != nil {
		return err
	}
	return labels.EnsureRepoLabels(c.App.Writer, cfgDir)
}

func runLabelsList(c *cli.Context) error {
	_, cfgDir, err := resolvePaths(c)
	if err != nil {
		return err
	}
	cfg, err := labels.LoadFromConfigDir(cfgDir)
	if err != nil {
		return err
	}
	cat := c.String("category")
	var names []string
	switch cat {
	case "type":
		names = cfg.TypeLabelNames()
	case "exception":
		names = cfg.ExceptionLabelNames()
	case "scope":
		names = cfg.IssueOnlyLabelNames()
	case "all":
		for _, r := range cfg.AllRepoLabels() {
			names = append(names, r.Name)
		}
		sort.Strings(names)
	default:
		return fmt.Errorf("labels list: unknown --category %q (use type, exception, scope, all)", cat)
	}
	out := map[string]any{
		"category": cat,
		"names":    names,
	}
	enc := json.NewEncoder(c.App.Writer)
	enc.SetIndent("", "  ")
	return enc.Encode(out)
}

func runLabelsSync(c *cli.Context) error {
	_, cfgDir, err := resolvePaths(c)
	if err != nil {
		return err
	}
	dry := c.Bool("dry-run")
	return labels.SyncRepoLabels(c.App.Writer, cfgDir, dry)
}

func newLabelsCategoryFlag() cli.Flag {
	return &cli.StringFlag{
		Name:    "category",
		Aliases: []string{"c"},
		Value:   "type",
		Usage:   "Label category: type | exception | scope | all",
	}
}

func newDryRunFlag() cli.Flag {
	return &cli.BoolFlag{
		Name:  "dry-run",
		Usage: "Print planned sync actions without calling gh",
	}
}
