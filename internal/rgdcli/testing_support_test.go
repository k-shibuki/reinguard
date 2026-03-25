package rgdcli

import (
	"testing"

	"github.com/urfave/cli/v2"
)

// testApp returns NewApp with ExitErrHandler set so cli.Exit does not call os.Exit
// during tests (urfave/cli default would terminate the test binary).
func testApp(t testing.TB, version string) *cli.App {
	t.Helper()
	app := NewApp(version)
	app.ExitErrHandler = func(*cli.Context, error) {}
	return app
}
