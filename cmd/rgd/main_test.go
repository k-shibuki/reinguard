package main

import (
	"bytes"
	"errors"
	"testing"

	"github.com/k-shibuki/reinguard/internal/rgdcli"
	"github.com/urfave/cli/v2"
)

func TestRun_version(t *testing.T) {
	t.Parallel()
	cases := []struct {
		name string
		args []string
	}{
		{name: "subcommand", args: []string{"rgd", "version"}},
		{name: "flag", args: []string{"rgd", "--version"}},
	}
	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			var buf bytes.Buffer
			app := rgdcli.NewApp("testver")
			app.ExitErrHandler = func(*cli.Context, error) {}
			app.Writer = &buf
			app.ErrWriter = &buf
			if err := app.Run(tc.args); err != nil {
				var ec cli.ExitCoder
				if errors.As(err, &ec) && ec.ExitCode() == 0 {
					// e.g. --version path returns cli.Exit("", 0)
				} else {
					t.Fatal(err)
				}
			}
			if !bytes.Contains(buf.Bytes(), []byte("testver")) {
				t.Fatalf("expected output to contain %q, got: %s", "testver", buf.String())
			}
		})
	}
}
