package scripttest

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

type resumeScriptStatus struct {
	LastIteration struct {
		StateID    string `json:"state_id"`
		RouteID    string `json:"route_id"`
		RecordedAt string `json:"recorded_at"`
	} `json:"last_iteration"`
	Terminal struct {
		Reason     string `json:"reason"`
		Summary    string `json:"summary"`
		RecordedAt string `json:"recorded_at"`
	} `json:"terminal"`
	ArtifactPath   string `json:"artifact_path"`
	Status         string `json:"status"`
	Reason         string `json:"reason"`
	Branch         string `json:"branch"`
	CurrentBranch  string `json:"current_branch"`
	ApprovalAt     string `json:"approval_recorded_at"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
	IssueNumber    int    `json:"issue_number"`
	ResumeEligible bool   `json:"resume_eligible"`
}

func TestAdapterRgdNextResumeScript_ShellSyntax(t *testing.T) {
	t.Parallel()

	root := repoRoot(t)
	paths := []string{
		filepath.Join(root, ".reinguard", "scripts", "adapter-rgd-next-resume.sh"),
		filepath.Join(root, ".reinguard", "scripts", "lib", "json_minimal.sh"),
	}
	for _, path := range paths {
		path := path
		t.Run(filepath.Base(path), func(t *testing.T) {
			t.Parallel()
			out, err := exec.Command("bash", "-n", path).CombinedOutput()
			if err != nil {
				t.Fatalf("bash -n %s: %v\n%s", path, err, out)
			}
		})
	}
}

func TestAdapterRgdNextResumeScript_ProposalThenApprove(t *testing.T) {
	t.Parallel()

	script := scriptPath(t, "adapter-rgd-next-resume.sh")
	repo := setupLocalReviewRepo(t)
	renameBranch(t, repo, "feature/104")

	startOut, err := runBashScript(t, repo, script, nil,
		"start",
		"--branch", "feature/104",
		"--issue", "104",
	)
	if err != nil {
		t.Fatalf("start: %v\n%s", err, startOut)
	}
	pendingOut, err := runBashScript(t, repo, script, nil, "status")
	if err != nil {
		t.Fatalf("status after start: %v\n%s", err, pendingOut)
	}
	var pending resumeScriptStatus
	unmarshalJSON(t, pendingOut, &pending)
	if pending.Status != "pending_approval" || pending.ResumeEligible || pending.ApprovalAt != "" {
		t.Fatalf("expected pending_approval before approve, got %+v", pending)
	}

	approveOut, err := runBashScript(t, repo, script, nil, "approve")
	if err != nil {
		t.Fatalf("approve: %v\n%s", err, approveOut)
	}
	activeOut, err := runBashScript(t, repo, script, nil, "status")
	if err != nil {
		t.Fatalf("status after approve: %v\n%s", err, activeOut)
	}
	var active resumeScriptStatus
	unmarshalJSON(t, activeOut, &active)
	if active.Status != "active" || !active.ResumeEligible || active.ApprovalAt == "" {
		t.Fatalf("expected active after approve, got %+v", active)
	}
}

func TestAdapterRgdNextResumeScript_Lifecycle(t *testing.T) {
	t.Parallel()

	script := scriptPath(t, "adapter-rgd-next-resume.sh")
	repo := setupLocalReviewRepo(t)
	renameBranch(t, repo, "feature/104")

	// Given: proposal artifact created before the approval gate, then approved Execute
	startOut, err := runBashScript(t, repo, script, nil,
		"start",
		"--branch", "feature/104",
		"--issue", "104",
		"--summary", "approved execute path",
	)
	if err != nil {
		t.Fatalf("start: %v\n%s", err, startOut)
	}

	// When: the adapter records progress and then a terminal allowed stop
	got := runResumeLifecycleStatus(t, repo, script)

	// Then: the artifact is stored under repo-local .reinguard/local and remains auditable
	if got.Status != "allowed_stop" || got.ResumeEligible {
		t.Fatalf("status = %+v", got)
	}
	if got.Branch != "feature/104" || got.IssueNumber != 104 {
		t.Fatalf("identity = %+v", got)
	}
	if got.LastIteration.StateID != "waiting_ci" || got.LastIteration.RouteID != "user-wait-ci" {
		t.Fatalf("last_iteration = %+v", got.LastIteration)
	}
	if got.Terminal.Reason != "tooling_session_limit" || got.Terminal.Summary != "context limit reached" {
		t.Fatalf("terminal = %+v", got.Terminal)
	}
	wantPath := filepath.Join(repo, ".reinguard", "local", "adapter", "rgd-next", "execute-resume.json")
	if got.ArtifactPath != wantPath {
		t.Fatalf("artifact_path = %q, want %q", got.ArtifactPath, wantPath)
	}
	if fi, statErr := os.Stat(wantPath); statErr != nil || fi.IsDir() {
		t.Fatalf("expected artifact file %q, stat err=%v", wantPath, statErr)
	}
}

func TestAdapterRgdNextResumeScript_StatusStaleOnDetachedHead(t *testing.T) {
	t.Parallel()

	script := scriptPath(t, "adapter-rgd-next-resume.sh")
	repo := setupLocalReviewRepo(t)
	renameBranch(t, repo, "feature/104")
	gitEmptyCommit(t, repo)

	startOut, err := runBashScript(t, repo, script, nil,
		"start",
		"--branch", "feature/104",
		"--issue", "104",
	)
	if err != nil {
		t.Fatalf("start: %v\n%s", err, startOut)
	}

	detachOut, err := exec.Command("git", "-C", repo, "checkout", "--detach").CombinedOutput()
	if err != nil {
		t.Fatalf("git checkout --detach: %v\n%s", err, detachOut)
	}

	statusOut, err := runBashScript(t, repo, script, nil, "status")
	if err != nil {
		t.Fatalf("status: %v\n%s", err, statusOut)
	}
	var got resumeScriptStatus
	unmarshalJSON(t, statusOut, &got)
	if got.Status != "stale" || got.ResumeEligible {
		t.Fatalf("status = %+v", got)
	}
	if got.Reason != "detached HEAD" {
		t.Fatalf("reason = %q", got.Reason)
	}
}

func TestAdapterRgdNextResumeScript_SummaryRoundTripWithQuotes(t *testing.T) {
	t.Parallel()

	script := scriptPath(t, "adapter-rgd-next-resume.sh")
	repo := setupLocalReviewRepo(t)
	renameBranch(t, repo, "feature/104")
	summary := `say "hi" and \n`

	startOut, err := runBashScript(t, repo, script, nil,
		"start",
		"--branch", "feature/104",
		"--issue", "104",
		"--summary", summary,
	)
	if err != nil {
		t.Fatalf("start: %v\n%s", err, startOut)
	}
	approveOut, err := runBashScript(t, repo, script, nil, "approve")
	if err != nil {
		t.Fatalf("approve: %v\n%s", err, approveOut)
	}
	showOut, err := runBashScript(t, repo, script, nil, "show")
	if err != nil {
		t.Fatalf("show: %v\n%s", err, showOut)
	}
	var artifact map[string]any
	unmarshalJSON(t, showOut, &artifact)
	got, _ := artifact["summary"].(string)
	if got != summary {
		t.Fatalf("summary round-trip: got %q want %q", got, summary)
	}
}

func TestAdapterRgdNextResumeScript_StatusStaleOnBranchMismatch(t *testing.T) {
	t.Parallel()

	script := scriptPath(t, "adapter-rgd-next-resume.sh")
	repo := setupLocalReviewRepo(t)
	renameBranch(t, repo, "main")

	// Given: a resume artifact recorded for a different branch
	startOut, err := runBashScript(t, repo, script, nil,
		"start",
		"--branch", "feature/104",
		"--issue", "104",
	)
	if err != nil {
		t.Fatalf("start: %v\n%s", err, startOut)
	}

	// When: the adapter checks status from the current branch
	statusOut, err := runBashScript(t, repo, script, nil, "status")
	if err != nil {
		t.Fatalf("status: %v\n%s", err, statusOut)
	}

	// Then: resume is blocked as stale rather than silently reused
	var got resumeScriptStatus
	unmarshalJSON(t, statusOut, &got)
	if got.Status != "stale" || got.ResumeEligible {
		t.Fatalf("status = %+v", got)
	}
	if got.Reason != "branch mismatch" || got.CurrentBranch != "main" {
		t.Fatalf("stale metadata = %+v", got)
	}
}

func TestAdapterRgdNextResumeScript_StatusInvalidForMalformedArtifact(t *testing.T) {
	t.Parallel()

	script := scriptPath(t, "adapter-rgd-next-resume.sh")
	repo := setupLocalReviewRepo(t)
	renameBranch(t, repo, "feature/104")
	artifactPath := filepath.Join(repo, ".reinguard", "local", "adapter", "rgd-next", "execute-resume.json")
	if err := os.MkdirAll(filepath.Dir(artifactPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(artifactPath, []byte(`{"status":"active"}`), 0o644); err != nil {
		t.Fatal(err)
	}

	// Given: a malformed adapter-local artifact
	// When: the status command validates it
	statusOut, err := runBashScript(t, repo, script, nil, "status")
	if err != nil {
		t.Fatalf("status: %v\n%s", err, statusOut)
	}

	// Then: the script fails closed with invalid status details
	var got resumeScriptStatus
	unmarshalJSON(t, statusOut, &got)
	if got.Status != "invalid" || got.ResumeEligible {
		t.Fatalf("status = %+v", got)
	}
	if !strings.Contains(got.Reason, "missing required keys") {
		t.Fatalf("reason = %q", got.Reason)
	}
}

func TestAdapterRgdNextResumeScript_FinishValidatesTerminalReason(t *testing.T) {
	t.Parallel()

	script := scriptPath(t, "adapter-rgd-next-resume.sh")
	repo := setupLocalReviewRepo(t)
	renameBranch(t, repo, "feature/104")

	// Given/When/Then: invalid finish combinations are rejected by the adapter-local contract.
	tests := []struct {
		name       string
		argsLine   string
		wantSubstr string
	}{
		{
			name:       "doneRequiresDodSatisfied",
			argsLine:   "finish --status done --reason tooling_session_limit",
			wantSubstr: "done status requires reason dod_satisfied",
		},
		{
			name:       "revokedRequiresScopeRevoked",
			argsLine:   "finish --status revoked --reason hard_stop",
			wantSubstr: "revoked status requires reason scope_revoked",
		},
		{
			name:       "allowedStopRejectsDodSatisfied",
			argsLine:   "finish --status allowed_stop --reason dod_satisfied",
			wantSubstr: "allowed_stop requires hard_stop, cannot_proceed, or tooling_session_limit",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Cleanup(func() {
				clearOut, clearErr := runBashScript(t, repo, script, nil, "clear")
				if clearErr != nil {
					t.Errorf("clear: %v\n%s", clearErr, clearOut)
				}
			})

			startOut, err := runBashScript(t, repo, script, nil,
				"start",
				"--branch", "feature/104",
				"--issue", "104",
			)
			if err != nil {
				t.Fatalf("start: %v\n%s", err, startOut)
			}

			out, err := runBashScript(t, repo, script, nil, strings.Fields(tt.argsLine)...)
			if err == nil {
				t.Fatalf("expected failure, got success:\n%s", out)
			}
			if !strings.Contains(out, tt.wantSubstr) {
				t.Fatalf("expected output to contain %q, got:\n%s", tt.wantSubstr, out)
			}
		})
	}
}

func runResumeLifecycleStatus(t *testing.T, repo, script string) resumeScriptStatus {
	t.Helper()

	approveOut, err := runBashScript(t, repo, script, nil, "approve")
	if err != nil {
		t.Fatalf("approve: %v\n%s", err, approveOut)
	}

	updateOut, err := runBashScript(t, repo, script, nil,
		"update",
		"--state-id", "waiting_ci",
		"--route-id", "user-wait-ci",
	)
	if err != nil {
		t.Fatalf("update: %v\n%s", err, updateOut)
	}
	finishOut, err := runBashScript(t, repo, script, nil,
		"finish",
		"--status", "allowed_stop",
		"--reason", "tooling_session_limit",
		"--summary", "context limit reached",
	)
	if err != nil {
		t.Fatalf("finish: %v\n%s", err, finishOut)
	}
	statusOut, err := runBashScript(t, repo, script, nil, "status")
	if err != nil {
		t.Fatalf("status: %v\n%s", err, statusOut)
	}

	var got resumeScriptStatus
	unmarshalJSON(t, statusOut, &got)
	return got
}

func renameBranch(t *testing.T, repo, branch string) {
	t.Helper()
	cmd := exec.Command("git", "branch", "-M", branch)
	cmd.Dir = repo
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git branch -M: %v\n%s", err, out)
	}
}

func gitEmptyCommit(t *testing.T, repo string) {
	t.Helper()
	cmd := exec.Command("git", "-C", repo, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git commit: %v\n%s", err, out)
	}
}

func unmarshalJSON(t *testing.T, raw string, target any) {
	t.Helper()
	if err := json.Unmarshal([]byte(raw), target); err != nil {
		t.Fatalf("unmarshal JSON: %v\n%s", err, raw)
	}
}
