package rgdcli

import "github.com/urfave/cli/v2"

// Urfave/cli mutates Flag implementations during parsing (e.g. BoolFlag.Apply).
// The same *cli.BoolFlag or *cli.StringFlag must not appear in more than one
// command's Flags slice (or both App.Flags and a subcommand). Sharing pointers
// causes data races when multiple App.Run calls run concurrently under -race.
// Always allocate a fresh instance per Flags slice via these factories.

// newConfigDirFlag returns a fresh --config-dir flag bound to REINGUARD_CONFIG_DIR.
func newConfigDirFlag() *cli.StringFlag {
	return &cli.StringFlag{Name: "config-dir", EnvVars: []string{"REINGUARD_CONFIG_DIR"}}
}

func newCwdFlag() *cli.StringFlag {
	return &cli.StringFlag{Name: "cwd", Usage: "working directory for git/gh"}
}

func newSerialFlag() *cli.BoolFlag {
	return &cli.BoolFlag{Name: "serial", Usage: "run observation providers sequentially"}
}

func newBranchFlag() *cli.StringFlag {
	return &cli.StringFlag{Name: "branch", Usage: "observe GitHub PR scope for this branch instead of the local checkout"}
}

func newPRNumberFlag() *cli.IntFlag {
	return &cli.IntFlag{Name: "pr", Usage: "observe GitHub facets for this pull request number"}
}

func newFailOnNonResolvedFlag() *cli.BoolFlag {
	return &cli.BoolFlag{
		Name:  "fail-on-non-resolved",
		Usage: "exit 2 for ambiguous, degraded, or unsupported outcomes where applicable",
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

func newGateStatusFlag() *cli.StringFlag {
	return &cli.StringFlag{Name: "status", Required: true, Usage: "recorded gate status (pass or fail)"}
}

func newGateChecksFileFlag() *cli.StringFlag {
	return &cli.StringFlag{Name: "checks-file", Usage: "JSON file containing a gate check array; use - to read the same array format from stdin"}
}

func newGateCheckFlag() *cli.StringSliceFlag {
	return &cli.StringSliceFlag{Name: "check", Usage: "inline check as id:status:summary (repeatable; alternative to --checks-file)"}
}

func newGateCheckJSONFlag() *cli.StringFlag {
	return &cli.StringFlag{Name: "check-json", Usage: "inline JSON for gate checks (one object or an array; supports evidence without a temp file)"}
}

func newGateInputsFileFlag() *cli.StringFlag {
	return &cli.StringFlag{Name: "inputs-file", Usage: "JSON file containing upstream gate proof inputs"}
}

func newGateInputGateFlag() *cli.StringSliceFlag {
	return &cli.StringSliceFlag{Name: "input-gate", Usage: "fresh passing gate id to copy into inputs (repeatable)"}
}

func newGateProducerProcedureFlag() *cli.StringFlag {
	return &cli.StringFlag{Name: "producer-procedure", Required: true, Usage: "procedure that recorded this gate artifact"}
}

func newGateProducerToolFlag() *cli.StringFlag {
	return &cli.StringFlag{Name: "producer-tool", Value: "rgd gate record", Usage: "tool or command that recorded this gate artifact"}
}

func newSchemaExportDirFlag() *cli.StringFlag {
	return &cli.StringFlag{Name: "dir", Aliases: []string{"d"}, Value: "schema-export"}
}

// observeFlags returns flags for any observe-shaped command.
func observeFlags() []cli.Flag {
	return []cli.Flag{
		newSerialFlag(),
		newCwdFlag(),
		newConfigDirFlag(),
		newBranchFlag(),
		newPRNumberFlag(),
		newFailOnNonResolvedFlag(),
	}
}

// newHelpFlag returns a fresh help flag instance. urfave/cli keeps its default
// help flag as a package global, so callers must avoid sharing pointers across
// App instances or command flag slices.
func newHelpFlag() *cli.BoolFlag {
	return &cli.BoolFlag{
		Name:               "help",
		Aliases:            []string{"h"},
		Usage:              "show help",
		DisableDefaultText: true,
	}
}

// Root-only clone of urfave's default VersionFlag. The library keeps that as a
// package global; appending it to multiple App instances causes data races
// under concurrent Run. HideVersion on the App plus this per-NewApp flag
// preserves behavior without sharing *BoolFlag with other Apps.
func newRootVersionFlag() *cli.BoolFlag {
	return &cli.BoolFlag{
		Name:               "version",
		Aliases:            []string{"v"},
		Usage:              "print the version",
		DisableDefaultText: true,
	}
}

// addHelpFlagOnCommands stops urfave from appending package-global cli.HelpFlag to
// each subcommand (which would race under concurrent App.Run). Users still get
// top-level help via root newRootHelpFlag and subcommand help via fresh
// per-command help flags, without sharing cli.HelpFlag across App instances.
func addHelpFlagOnCommands(cmds []*cli.Command) {
	for _, c := range cmds {
		c.Flags = append(c.Flags, newHelpFlag())
		c.HideHelp = true
		addHelpFlagOnCommands(c.Subcommands)
	}
}
