// Package main provides the rgd CLI entrypoint (Substrate: docs/cli.md, ADR-0001).
package main

import (
	"errors"
	"log"
	"os"

	"github.com/k-shibuki/reinguard/internal/rgdcli"
	"github.com/urfave/cli/v2"
)

// Set at link time: go build -ldflags "-X main.version=1.2.3".
var version = "dev"

// main runs the urfave/cli app, logs non-empty error messages to stderr,
// and exits with the mapped status code.
func main() {
	if err := run(os.Args, version); err != nil {
		exitCode, message := exitStatus(err)
		if message != "" {
			log.Print(message)
		}
		os.Exit(exitCode)
	}
}

// run wires argv and version into [rgdcli.NewApp] and runs the CLI.
func run(args []string, ver string) error {
	app := rgdcli.NewApp(ver)
	return app.Run(args)
}

// exitStatus extracts an exit code and message from err. It prefers
// [cli.ExitCoder], then falls back to any error exposing ExitCode(), and uses
// exit code 1 for other errors. Nil returns (0, "").
func exitStatus(err error) (int, string) {
	if err == nil {
		return 0, ""
	}
	var cliExiter cli.ExitCoder
	if errors.As(err, &cliExiter) {
		return cliExiter.ExitCode(), cliExiter.Error()
	}

	type exitCoder interface {
		ExitCode() int
	}
	var exiter exitCoder
	if errors.As(err, &exiter) {
		return exiter.ExitCode(), err.Error()
	}
	return 1, err.Error()
}
