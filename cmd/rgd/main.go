// Package main provides the rgd CLI entrypoint (Substrate: docs/cli.md, ADR-0001).
package main

import (
	"log"
	"os"

	"github.com/k-shibuki/reinguard/internal/rgdcli"
)

// Set at link time: go build -ldflags "-X main.version=1.2.3".
var version = "dev"

// main runs the urfave/cli app and logs fatal errors to stderr.
func main() {
	if err := run(os.Args, version); err != nil {
		log.Fatal(err)
	}
}

// run constructs the rgd app with ver and executes args.
func run(args []string, ver string) error {
	app := rgdcli.NewApp(ver)
	return app.Run(args)
}
