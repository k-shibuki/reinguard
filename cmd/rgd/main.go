// Command rgd is the reinguard CLI.
package main

import (
	"log"
	"os"

	"github.com/k-shibuki/reinguard/internal/rgdcli"
)

// Set at link time: go build -ldflags "-X main.version=1.2.3".
var version = "dev"

func main() {
	if err := run(os.Args, version); err != nil {
		log.Fatal(err)
	}
}

func run(args []string, ver string) error {
	app := rgdcli.NewApp(ver)
	return app.Run(args)
}
