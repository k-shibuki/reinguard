package rgdcli

import (
	"flag"
	"strconv"
	"strings"
	"testing"

	"github.com/urfave/cli/v2"
)

func TestParseObserveScopeFlags(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name               string
		branch             string
		wantBranch         string
		wantErrSubstr      string
		observationFileSet bool
		setPR              bool
		pr                 int
		wantPR             int
	}{
		{
			name:          "given no scope flags when live observe then zero values returned",
			wantBranch:    "",
			wantPR:        0,
			wantErrSubstr: "",
		},
		{
			name:          "given branch when live observe then branch returned",
			branch:        "main",
			wantBranch:    "main",
			wantPR:        0,
			wantErrSubstr: "",
		},
		{
			name:          "given pr when live observe then pr returned",
			setPR:         true,
			pr:            5,
			wantBranch:    "",
			wantPR:        5,
			wantErrSubstr: "",
		},
		{
			name:               "given branch with observation file then error",
			branch:             "main",
			observationFileSet: true,
			wantErrSubstr:      "--branch/--pr cannot be used with --observation-file",
		},
		{
			name:               "given pr with observation file then error",
			setPR:              true,
			pr:                 5,
			observationFileSet: true,
			wantErrSubstr:      "--branch/--pr cannot be used with --observation-file",
		},
		{
			name:          "given branch and pr when live observe then both values preserved",
			branch:        "topic",
			setPR:         true,
			pr:            9,
			wantBranch:    "topic",
			wantPR:        9,
			wantErrSubstr: "",
		},
		{
			name:          "given zero pr then error",
			setPR:         true,
			pr:            0,
			wantErrSubstr: "--pr must be greater than 0",
		},
		{
			name:          "given negative pr then error",
			setPR:         true,
			pr:            -1,
			wantErrSubstr: "--pr must be greater than 0",
		},
		{
			name:          "given whitespace branch then error",
			branch:        "   ",
			wantErrSubstr: "--branch must be non-empty",
		},
	}
	for _, tc := range tests {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			// Given: a CLI context with the scope flags registered
			set := flag.NewFlagSet("test", flag.ContinueOnError)
			for _, fl := range []cli.Flag{newBranchFlag(), newPRNumberFlag()} {
				if err := fl.Apply(set); err != nil {
					t.Fatal(err)
				}
			}
			if err := set.Parse(nil); err != nil {
				t.Fatal(err)
			}
			if tc.branch != "" {
				if err := set.Set("branch", tc.branch); err != nil {
					t.Fatal(err)
				}
			}
			if tc.setPR {
				if err := set.Set("pr", strconv.Itoa(tc.pr)); err != nil {
					t.Fatal(err)
				}
			}
			ctx := cli.NewContext(cli.NewApp(), set, nil)

			// When: parseObserveScopeFlags validates the selected flags
			gotBranch, gotPR, err := parseObserveScopeFlags(ctx, tc.observationFileSet)

			// Then: the parsed values or validation error match the contract
			if tc.wantErrSubstr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErrSubstr) {
					t.Fatalf("expected error containing %q, got %v", tc.wantErrSubstr, err)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if gotBranch != tc.wantBranch || gotPR != tc.wantPR {
				t.Fatalf("got branch=%q pr=%d want branch=%q pr=%d", gotBranch, gotPR, tc.wantBranch, tc.wantPR)
			}
		})
	}
}
