package scripttest

import (
	"strings"
	"testing"
)

const validPRBody = "## Summary\n\n" +
	"- Explain why the change exists.\n\n" +
	"## Traceability\n\n" +
	"Closes #97\n\n" +
	"Refs: #59\n\n" +
	"## Definition of Done\n\n" +
	"- [x] Tests added or updated (`go test ./...`)\n" +
	"- [x] `go vet ./...` clean\n" +
	"- [x] Lint clean (golangci-lint / CI)\n" +
	"- [x] Documentation updated if behavior or public CLI surface changed\n\n" +
	"## Test plan\n\n" +
	"1. Run `go test ./...`\n" +
	"2. Run `go vet ./...`\n\n" +
	"## Risk / Impact\n\n" +
	"- Affects repository-local workflow docs and checks.\n\n" +
	"## Rollback Plan\n\n" +
	"- Revert the workflow commit if the local gate blocks PR creation incorrectly.\n\n" +
	"## Exception\n\n" +
	"- Type:\n" +
	"- Justification:\n"

func TestCheckPRPolicyScript(t *testing.T) {
	// mustMikefarahYq uses t.Setenv, so this test must stay serial.
	root := repoRoot(t)
	mustMikefarahYq(t, root)
	script := scriptPath(t, "check-pr-policy.sh")

	mustReplaceOne := func(old, repl string) string {
		t.Helper()
		out := strings.Replace(validPRBody, old, repl, 1)
		if out == validPRBody {
			t.Fatalf("fixture setup failed: replacement target not found: %q", old)
		}
		return out
	}

	// Given/When/Then: PR body/title/label fixtures are validated against repository PR policy.
	tests := []struct {
		name          string
		title         string
		body          string
		base          string
		labels        []string
		bodyFile      string
		wantSubstr    []string
		wantErr       bool
		skipTitle     bool
		emptyBodyFile bool
	}{
		{
			name:       "validPRBody",
			title:      "fix(workflow): add script integration checks",
			body:       validPRBody,
			labels:     []string{"fix"},
			base:       "main",
			wantSubstr: []string{"PR policy pre-flight OK."},
		},
		{
			name:       "missingIssueLink",
			title:      "fix(workflow): add script integration checks",
			body:       mustReplaceOne("Closes #97", ""),
			labels:     []string{"fix"},
			base:       "main",
			wantErr:    true,
			wantSubstr: []string{"body must contain 'Closes #N'"},
		},
		{
			name:       "wrongBaseBranch",
			title:      "fix(workflow): add script integration checks",
			body:       validPRBody,
			labels:     []string{"fix"},
			base:       "feat/stacked",
			wantErr:    true,
			wantSubstr: []string{"PR must target main"},
		},
		{
			name:       "missingTypeLabel",
			title:      "fix(workflow): add script integration checks",
			body:       validPRBody,
			base:       "main",
			wantErr:    true,
			wantSubstr: []string{"--label <type>"},
		},
		{
			name:       "emptyTypeLabelValue",
			title:      "fix(workflow): add script integration checks",
			body:       validPRBody,
			base:       "main",
			labels:     []string{""},
			wantErr:    true,
			wantSubstr: []string{"--label requires a non-empty value"},
		},
		{
			name:       "labelsPresentButNoTypeLabel",
			title:      "fix(workflow): add script integration checks",
			body:       validPRBody,
			base:       "main",
			labels:     []string{"meta"},
			wantErr:    true,
			wantSubstr: []string{"Type label: must have exactly one type label. Got none."},
		},
		{
			name:       "masterBaseRejected",
			title:      "fix(workflow): add script integration checks",
			body:       validPRBody,
			labels:     []string{"fix"},
			base:       "master",
			wantErr:    true,
			wantSubstr: []string{"PR must target main. Got: master"},
		},
		{
			name:       "testPlanAllCapsHeading",
			title:      "fix(workflow): add script integration checks",
			body:       mustReplaceOne("## Test plan\n", "## TEST PLAN\n"),
			labels:     []string{"fix"},
			base:       "main",
			wantSubstr: []string{"PR policy pre-flight OK."},
		},
		{
			name:       "emptyTestPlanBody",
			title:      "fix(workflow): add script integration checks",
			body:       mustReplaceOne("## Test plan\n\n1. Run `go test ./...`\n2. Run `go vet ./...`\n\n", "## Test plan\n\n## Risk / Impact\n\n"),
			labels:     []string{"fix"},
			base:       "main",
			wantErr:    true,
			wantSubstr: []string{"Test plan: section exists but appears empty."},
		},
		{
			name:       "commentOnlyRiskImpact",
			title:      "fix(workflow): add script integration checks",
			body:       mustReplaceOne("- Affects repository-local workflow docs and checks.\n\n", "<!-- placeholder -->\n\n"),
			labels:     []string{"fix"},
			base:       "main",
			wantErr:    true,
			wantSubstr: []string{"Risk / Impact: section exists but appears empty."},
		},
		{
			name:       "commentOnlyRollbackPlan",
			title:      "fix(workflow): add script integration checks",
			body:       mustReplaceOne("- Revert the workflow commit if the local gate blocks PR creation incorrectly.\n\n", "<!-- placeholder -->\n\n"),
			labels:     []string{"fix"},
			base:       "main",
			wantErr:    true,
			wantSubstr: []string{"Rollback Plan: section exists but appears empty."},
		},
		{
			name:       "missingBodyFile",
			title:      "fix(workflow): add script integration checks",
			base:       "main",
			labels:     []string{"fix"},
			bodyFile:   "does-not-exist.md",
			wantErr:    true,
			wantSubstr: []string{"body file not found"},
		},
		{
			name:       "emptyTitleFlag",
			title:      "",
			body:       validPRBody,
			labels:     []string{"fix"},
			base:       "main",
			wantErr:    true,
			wantSubstr: []string{"--title requires a non-empty value"},
		},
		{
			name:          "emptyBodyFileFlag",
			title:         "fix(workflow): add script integration checks",
			body:          validPRBody,
			labels:        []string{"fix"},
			base:          "main",
			wantErr:       true,
			wantSubstr:    []string{"--body-file requires a non-empty path"},
			emptyBodyFile: true,
		},
		{
			name:       "emptyBaseFlag",
			title:      "fix(workflow): add script integration checks",
			body:       validPRBody,
			labels:     []string{"fix"},
			base:       "",
			wantErr:    true,
			wantSubstr: []string{"--base requires a non-empty branch name"},
		},
		{
			name:       "missingTitleShowsUsage",
			title:      "fix(workflow): add script integration checks",
			body:       validPRBody,
			labels:     []string{"fix"},
			base:       "main",
			wantErr:    true,
			wantSubstr: []string{"Usage: check-pr-policy.sh"},
			skipTitle:  true,
		},
		{
			name:       "commentOnlyTestPlanAfterStrip",
			title:      "fix(workflow): add script integration checks",
			body:       mustReplaceOne("## Test plan\n\n1. Run `go test ./...`\n2. Run `go vet ./...`\n\n", "## Test plan\n\n<!-- no substantive test plan -->\n\n"),
			labels:     []string{"fix"},
			base:       "main",
			wantErr:    true,
			wantSubstr: []string{"Test plan: section exists but appears empty."},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			bodyFile := tt.bodyFile
			if bodyFile == "" && !tt.emptyBodyFile {
				bodyFile = writeTempFile(t, t.TempDir(), "pr-body-*.md", tt.body)
			}
			var args []string
			switch {
			case tt.skipTitle:
				args = []string{"--body-file", bodyFile, "--base", tt.base}
				for _, label := range tt.labels {
					args = append(args, "--label", label)
				}
			case tt.emptyBodyFile:
				args = []string{"--title", tt.title, "--body-file", "", "--base", tt.base}
				for _, label := range tt.labels {
					args = append(args, "--label", label)
				}
			default:
				args = []string{"--title", tt.title, "--body-file", bodyFile, "--base", tt.base}
				for _, label := range tt.labels {
					args = append(args, "--label", label)
				}
			}

			out, err := runBashScript(t, root, script, nil, args...)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got success:\n%s", out)
				}
			} else if err != nil {
				t.Fatalf("check-pr-policy: %v\n%s", err, out)
			}

			for _, sub := range tt.wantSubstr {
				if !strings.Contains(out, sub) {
					t.Fatalf("expected output to contain %q, got:\n%s", sub, out)
				}
			}
		})
	}
}
