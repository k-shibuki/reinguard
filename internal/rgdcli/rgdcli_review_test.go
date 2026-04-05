package rgdcli

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/urfave/cli/v2"
)

func TestReviewBodyFromFlags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name          string
		body          string
		fileBody      string
		wantBody      string
		wantErrSubstr string
		useBodyFile   bool
	}{
		{
			name:     "given body when parsing then inline body returned",
			body:     "Fixed in latest commit.",
			wantBody: "Fixed in latest commit.",
		},
		{
			name:          "given whitespace body when parsing then required error",
			body:          "   ",
			wantErrSubstr: "required",
		},
		{
			name:        "given body file when parsing then file body returned",
			fileBody:    "By design. Scoped CI follows the observed PR head.\n",
			wantBody:    "By design. Scoped CI follows the observed PR head.\n",
			useBodyFile: true,
		},
		{
			name:          "given body and file when parsing then error",
			body:          "x",
			fileBody:      "x",
			wantErrSubstr: "mutually exclusive",
			useBodyFile:   true,
		},
		{
			name:          "given empty body file when parsing then error",
			fileBody:      "   \n",
			wantErrSubstr: "non-empty",
			useBodyFile:   true,
		},
		{
			name:          "given no body flags when parsing then error",
			wantErrSubstr: "required",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			runReviewBodyCase(t, tc)
		})
	}
}

func runReviewBodyCase(t *testing.T, tc struct {
	name          string
	body          string
	fileBody      string
	wantBody      string
	wantErrSubstr string
	useBodyFile   bool
}) {
	t.Helper()
	// Given: a CLI context and optional reply body file
	wd := t.TempDir()
	if tc.useBodyFile {
		if err := os.WriteFile(filepath.Join(wd, "reply.md"), []byte(tc.fileBody), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	ctx := newReviewBodyContext(t, tc.body, tc.useBodyFile)

	// When: reviewBodyFromFlags reads and validates the selected input
	got, err := reviewBodyFromFlags(wd, ctx)

	// Then: the resulting body or validation error matches the command contract
	if tc.wantErrSubstr != "" {
		if err == nil || !strings.Contains(err.Error(), tc.wantErrSubstr) {
			t.Fatalf("expected error containing %q, got %v", tc.wantErrSubstr, err)
		}
		return
	}
	if err != nil {
		t.Fatal(err)
	}
	if got != tc.wantBody {
		t.Fatalf("got %q want %q", got, tc.wantBody)
	}
}

func newReviewBodyContext(t *testing.T, body string, useBodyFile bool) *cli.Context {
	t.Helper()
	set := flag.NewFlagSet("test", flag.ContinueOnError)
	for _, fl := range []cli.Flag{
		&cli.StringFlag{Name: "body"},
		&cli.StringFlag{Name: "body-file"},
	} {
		if err := fl.Apply(set); err != nil {
			t.Fatal(err)
		}
	}
	if err := set.Parse(nil); err != nil {
		t.Fatal(err)
	}
	if body != "" {
		if err := set.Set("body", body); err != nil {
			t.Fatal(err)
		}
	}
	if useBodyFile {
		if err := set.Set("body-file", "reply.md"); err != nil {
			t.Fatal(err)
		}
	}
	return cli.NewContext(cli.NewApp(), set, nil)
}
