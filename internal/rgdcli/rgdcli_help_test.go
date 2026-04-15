package rgdcli

import (
	"bytes"
	"strings"
	"testing"
)

func TestSubcommandsSupportHelpFlag(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		args []string
	}{
		{name: "config validate", args: []string{"config", "validate", "--help"}},
		{name: "schema export", args: []string{"schema", "export", "--help"}},
		{name: "observe", args: []string{"observe", "--help"}},
		{name: "observe workflow-position", args: []string{"observe", "workflow-position", "--help"}},
		{name: "observe git", args: []string{"observe", "git", "--help"}},
		{name: "observe github", args: []string{"observe", "github", "--help"}},
		{name: "observe github issues", args: []string{"observe", "github", "issues", "--help"}},
		{name: "observe github pull-requests", args: []string{"observe", "github", "pull-requests", "--help"}},
		{name: "observe github ci", args: []string{"observe", "github", "ci", "--help"}},
		{name: "observe github reviews", args: []string{"observe", "github", "reviews", "--help"}},
		{name: "state eval", args: []string{"state", "eval", "--help"}},
		{name: "route select", args: []string{"route", "select", "--help"}},
		{name: "knowledge index", args: []string{"knowledge", "index", "--help"}},
		{name: "knowledge pack", args: []string{"knowledge", "pack", "--help"}},
		{name: "context build", args: []string{"context", "build", "--help"}},
		{name: "gate record", args: []string{"gate", "record", "--help"}},
		{name: "gate status", args: []string{"gate", "status", "--help"}},
		{name: "gate show", args: []string{"gate", "show", "--help"}},
		{name: "guard eval", args: []string{"guard", "eval", "--help"}},
		{name: "review reply-thread", args: []string{"review", "reply-thread", "--help"}},
		{name: "review resolve-thread", args: []string{"review", "resolve-thread", "--help"}},
		{name: "ensure-labels", args: []string{"ensure-labels", "--help"}},
		{name: "labels list", args: []string{"labels", "list", "--help"}},
		{name: "labels sync", args: []string{"labels", "sync", "--help"}},
		{name: "version", args: []string{"version", "--help"}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()

			var out bytes.Buffer
			app := NewApp("test")
			app.Writer = &out
			app.ErrWriter = &out

			args := append([]string{"rgd"}, tc.args...)
			if err := app.Run(args); err != nil {
				t.Fatalf("unexpected help error: %v", err)
			}
			if !strings.Contains(out.String(), "NAME:") {
				t.Fatalf("expected help output for %v, got %q", args, out.String())
			}
		})
	}
}
