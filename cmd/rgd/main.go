// Package main provides the rgd CLI entrypoint (Substrate: docs/cli.md, ADR-0001).
package main

import (
	"fmt"
	"log"
	"os"

	"github.com/k-shibuki/reinguard/internal/rgdcli"
	"github.com/urfave/cli/v2"
)

// Set at link time: go build -ldflags "-X main.version=1.2.3".
var version = "dev"

// main runs the urfave/cli app and logs fatal errors to stderr.
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

func exitStatus(err error) (int, string) {
	if err == nil {
		return 0, ""
	}
	type exitCoder interface {
		ExitCode() int
	}
	if exiter, ok := err.(exitCoder); ok {
		return exiter.ExitCode(), err.Error()
	}
	if exiter, ok := err.(cli.ExitCoder); ok {
		return exiter.ExitCode(), exiter.Error()
	}
	return 1, fmt.Sprint(err)
}
