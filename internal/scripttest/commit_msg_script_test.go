package scripttest

import (
	"strings"
	"testing"
)

func TestCheckCommitMsgScript(t *testing.T) {
	root := repoRoot(t)
	mustMikefarahYq(t, root)
	script := scriptPath(t, "check-commit-msg.sh")

	// Given/When/Then: commit message fixtures are validated by the commit-msg shell hook.
	tests := []struct {
		name       string
		message    string
		wantSubstr []string
		wantErr    bool
	}{
		{
			name: "validStandardCommit",
			message: "fix(workflow): keep scripts testable\n\n" +
				"- keep shell wrapper thin\n\n" +
				"Refs: #97\n",
			wantSubstr: []string{},
		},
		{
			name: "missingRefsFails",
			message: "fix(workflow): keep scripts testable\n\n" +
				"- keep shell wrapper thin\n",
			wantErr:    true,
			wantSubstr: []string{"Missing 'Refs: #<number>'"},
		},
		{
			name:       "docsRequiresBody",
			message:    "docs(workflow): explain shell tooling\n",
			wantErr:    true,
			wantSubstr: []string{"docs commits must include justification in body"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeTempFile(t, t.TempDir(), "commit-msg-*.txt", tt.message)

			out, err := runBashScript(t, root, script, nil, path)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got success:\n%s", out)
				}
			} else if err != nil {
				t.Fatalf("check-commit-msg: %v\n%s", err, out)
			}

			for _, sub := range tt.wantSubstr {
				if !strings.Contains(out, sub) {
					t.Fatalf("expected output to contain %q, got:\n%s", sub, out)
				}
			}
		})
	}
}
