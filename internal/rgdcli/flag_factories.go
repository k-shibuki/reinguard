package rgdcli

import "github.com/urfave/cli/v2"

// Urfave/cli mutates Flag implementations during parsing (e.g. BoolFlag.Apply).
// The same *cli.BoolFlag or *cli.StringFlag must not appear in more than one
// command's Flags slice (or both App.Flags and a subcommand). Sharing pointers
// causes data races when multiple App.Run calls run concurrently under -race.
// Always allocate a fresh instance per Flags slice via these factories.

func newConfigDirFlag() *cli.StringFlag {
	return &cli.StringFlag{Name: "config-dir", EnvVars: []string{"REINGUARD_CONFIG_DIR"}}
}

func newCwdFlag() *cli.StringFlag {
	return &cli.StringFlag{Name: "cwd", Usage: "working directory for git/gh"}
}

func newSerialFlag() *cli.BoolFlag {
	return &cli.BoolFlag{Name: "serial", Usage: "run observation providers sequentially"}
}

func newFailOnNonResolvedFlag() *cli.BoolFlag {
	return &cli.BoolFlag{
		Name:  "fail-on-non-resolved",
		Usage: "exit non-zero for ambiguous/degraded outcomes where applicable",
	}
}

func newObservationFileFlag() *cli.StringFlag {
	return &cli.StringFlag{Name: "observation-file"}
}

func newObservationFileRequiredFlag() *cli.StringFlag {
	return &cli.StringFlag{Name: "observation-file", Required: true}
}

func newKnowledgeQueryFlag() *cli.StringFlag {
	return &cli.StringFlag{Name: "query", Usage: "case-insensitive substring match against entry triggers"}
}

func newStateFileFlag() *cli.StringFlag {
	return &cli.StringFlag{Name: "state-file"}
}

func newSchemaExportDirFlag() *cli.StringFlag {
	return &cli.StringFlag{Name: "dir", Aliases: []string{"d"}, Value: "schema-export"}
}

// observeFlags returns flags for any observe-shaped command.
func observeFlags() []cli.Flag {
	return []cli.Flag{newSerialFlag(), newCwdFlag(), newConfigDirFlag(), newFailOnNonResolvedFlag()}
}

// Root-only clones of urfave's default HelpFlag / VersionFlag. The library keeps
// those as package globals; appending them to multiple App instances causes
// data races under concurrent Run. HideHelp/HideVersion on the App plus these
// per-NewApp flags preserves behavior without sharing *BoolFlag with other Apps.
func newRootHelpFlag() *cli.BoolFlag {
	return &cli.BoolFlag{
		Name:               "help",
		Aliases:            []string{"h"},
		Usage:              "show help",
		DisableDefaultText: true,
	}
}

func newRootVersionFlag() *cli.BoolFlag {
	return &cli.BoolFlag{
		Name:               "version",
		Aliases:            []string{"v"},
		Usage:              "print the version",
		DisableDefaultText: true,
	}
}

// hideHelpOnCommands stops urfave from appending package-global cli.HelpFlag to
// each subcommand (which would race under concurrent App.Run). Users still get
// top-level help via root newRootHelpFlag; adding a fresh help BoolFlag per
// subcommand's Flags slice is the extension path for `subcmd --help`.
func hideHelpOnCommands(cmds []*cli.Command) {
	for _, c := range cmds {
		c.HideHelp = true
		hideHelpOnCommands(c.Subcommands)
	}
}
