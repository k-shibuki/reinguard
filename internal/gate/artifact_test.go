package gate

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestRecordAndShow_roundTrip(t *testing.T) {
	t.Parallel()
	// Given: a git repo on main and one passing local-verification proof
	repo := initGitRepo(t)
	cfgDir := t.TempDir()
	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)

	// When: Record writes the artifact and Show reads it back
	art, err := Record(context.Background(), cfgDir, repo, "local-verification", StatusPass, Producer{
		Procedure: "implement",
		Tool:      "rgd gate record",
	}, nil, []Check{
		{ID: "go-test", Status: StatusPass, Summary: "go test ./... -race", Evidence: "bash .reinguard/scripts/with-repo-local-state.sh -- go test ./... -race"},
	}, now)
	if err != nil {
		t.Fatal(err)
	}
	got, err := Show(cfgDir, "local-verification")
	if err != nil {
		t.Fatal(err)
	}

	// Then: the persisted artifact keeps stable metadata and checks
	if got.GateID != "local-verification" || got.Status != StatusPass {
		t.Fatalf("unexpected artifact: %+v", got)
	}
	if got.RecordedAt != now.Format(time.RFC3339) {
		t.Fatalf("recorded_at=%q want %q", got.RecordedAt, now.Format(time.RFC3339))
	}
	if len(got.Checks) != 1 || got.Checks[0].ID != "go-test" {
		t.Fatalf("checks=%+v", got.Checks)
	}
	if art.Subject.HeadSHA != got.Subject.HeadSHA || art.Subject.Branch != got.Subject.Branch {
		t.Fatalf("record/show mismatch: wrote=%+v got=%+v", art, got)
	}
	if got.Producer.Procedure != "implement" || got.Producer.Tool != "rgd gate record" {
		t.Fatalf("producer=%+v", got.Producer)
	}
}

func TestRecord_rejectsDetachedHEAD(t *testing.T) {
	t.Parallel()
	repo := initGitRepo(t)
	runGit(t, repo, "checkout", "--detach", "HEAD")
	cfgDir := t.TempDir()
	_, err := Record(context.Background(), cfgDir, repo, "local-verification", StatusPass, Producer{
		Procedure: "implement",
		Tool:      "rgd gate record",
	}, nil, []Check{
		{ID: "go-test", Status: StatusPass, Summary: "go test ./..."},
	}, time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("expected error on detached HEAD")
	}
	if !strings.Contains(err.Error(), "detached HEAD") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRecord_rejectsInvalidArtifactStatus(t *testing.T) {
	t.Parallel()
	repo := initGitRepo(t)
	cfgDir := t.TempDir()
	_, err := Record(context.Background(), cfgDir, repo, "local-verification", "bogus", Producer{
		Procedure: "implement",
		Tool:      "rgd gate record",
	}, nil, []Check{
		{ID: "go-test", Status: StatusPass, Summary: "go test ./..."},
	}, time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC))
	if err == nil {
		t.Fatal("expected error for invalid status")
	}
	if !strings.Contains(err.Error(), "status") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRecord_rejectsPRReadinessWithoutUpstreamProofs(t *testing.T) {
	t.Parallel()
	// Given: a git repo with no prior local-verification proof recorded
	repo := initGitRepo(t)
	cfgDir := t.TempDir()
	// When: Record pr-readiness without upstream input proofs
	_, err := Record(context.Background(), cfgDir, repo, "pr-readiness", StatusPass, Producer{
		Procedure: "change-inspect",
		Tool:      "rgd gate record",
	}, nil, []Check{
		{ID: "review-closure", Status: StatusPass, Summary: "all local findings classified and closed"},
	}, time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC))
	// Then: error references missing local-verification proof
	if err == nil {
		t.Fatal("expected error for missing local proof inputs")
	}
	if !strings.Contains(err.Error(), "local-verification") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRecord_rejectsLocalCodeRabbitWithoutRequiredCheck(t *testing.T) {
	t.Parallel()
	// Given: a git repo with no prior gate artifacts
	repo := initGitRepo(t)
	cfgDir := t.TempDir()
	// When: Record local-coderabbit without the required check id
	_, err := Record(context.Background(), cfgDir, repo, "local-coderabbit", StatusPass, Producer{
		Procedure: "change-inspect",
		Tool:      "rgd gate record",
	}, nil, []Check{
		{ID: "review-closure", Status: StatusPass, Summary: "wrong check"},
	}, time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC))
	// Then: error references missing local-coderabbit-cli check
	if err == nil {
		t.Fatal("expected error for missing local-coderabbit proof check")
	}
	if !strings.Contains(err.Error(), "local-coderabbit-cli") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRecord_allowsPRReadinessWhenPrePRAIReviewOptional(t *testing.T) {
	t.Parallel()
	// Given: config with pre_pr_ai_review.required=false and local-verification proof inputs
	repo := initGitRepo(t)
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(`schema_version: "0.6.0"
default_branch: main
workflow:
  runtime_gate_roles:
    pre_pr_ai_review:
      gate_id: local-coderabbit
      required: false
providers: []
`))
	subject := Subject{HeadSHA: currentHeadForTest(t, repo), Branch: currentBranchForTest(t, repo)}
	recordedAt := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC).Format(time.RFC3339)
	now := time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC)

	// And: pr-readiness still rejects missing local-verification (negative control for this config)
	_, errNeg := Record(context.Background(), cfgDir, repo, "pr-readiness", StatusPass, Producer{
		Procedure: "change-inspect",
		Tool:      "rgd gate record",
	}, nil, []Check{
		{ID: "review-closure", Status: StatusPass, Summary: "all local findings classified and closed"},
	}, now)
	if errNeg == nil || !strings.Contains(errNeg.Error(), "local-verification") {
		t.Fatalf("expected missing local-verification failure, got %v", errNeg)
	}

	// When: Record pr-readiness with only local-verification as upstream proof
	_, err := Record(context.Background(), cfgDir, repo, "pr-readiness", StatusPass, Producer{
		Procedure: "change-inspect",
		Tool:      "rgd gate record",
	}, []Input{
		{
			GateID:     "local-verification",
			Status:     StatusPass,
			Subject:    subject,
			RecordedAt: recordedAt,
		},
	}, []Check{
		{ID: "review-closure", Status: StatusPass, Summary: "all local findings classified and closed"},
	}, now)
	// Then: success without local-coderabbit input proof
	if err != nil {
		t.Fatalf("expected optional pre-PR AI review proof to be skippable, got %v", err)
	}
}

func TestStatus_classifiesArtifacts(t *testing.T) {
	t.Parallel()
	// Given/When/Then: gate status derives missing, invalid, stale, pass, and fail outcomes
	repo := initGitRepo(t)
	branch := currentBranchForTest(t, repo)
	head := currentHeadForTest(t, repo)
	subject := Subject{HeadSHA: head, Branch: branch}
	producer := Producer{Procedure: "implement", Tool: "rgd gate record"}

	tests := []struct {
		name       string
		gateID     string
		prepare    func(t *testing.T, cfgDir string)
		wantStatus string
		wantReason string
	}{
		{
			name:       "missing",
			gateID:     "local-verification",
			prepare:    func(t *testing.T, cfgDir string) {},
			wantStatus: StatusMissing,
			wantReason: "artifact missing",
		},
		{
			name:   "invalid",
			gateID: "local-verification",
			prepare: func(t *testing.T, cfgDir string) {
				path, err := ArtifactPath(cfgDir, "local-verification")
				if err != nil {
					t.Fatal(err)
				}
				writeFile(t, path, []byte(`{`))
			},
			wantStatus: StatusInvalid,
			wantReason: "parse",
		},
		{
			name:   "invalid gate id mismatch",
			gateID: "local-verification",
			prepare: func(t *testing.T, cfgDir string) {
				path, err := ArtifactPath(cfgDir, "local-verification")
				if err != nil {
					t.Fatal(err)
				}
				writeFile(t, path, marshalArtifactForTest(t, Artifact{
					SchemaVersion: "0.6.0",
					GateID:        "other-gate",
					Status:        StatusPass,
					RecordedAt:    time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC).Format(time.RFC3339),
					Subject:       subject,
					Producer:      producer,
					Inputs:        []Input{},
					Checks:        []Check{{ID: "go-test", Status: StatusPass, Summary: "go test ./..."}},
				}))
			},
			wantStatus: StatusInvalid,
			wantReason: `declares gate_id "other-gate"; want "local-verification"`,
		},
		{
			name:   "stale branch mismatch",
			gateID: "local-verification",
			prepare: func(t *testing.T, cfgDir string) {
				writeArtifactForTest(t, cfgDir, Artifact{
					SchemaVersion: "0.6.0",
					GateID:        "local-verification",
					Status:        StatusPass,
					RecordedAt:    time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC).Format(time.RFC3339),
					Subject:       Subject{HeadSHA: head, Branch: "other-branch"},
					Producer:      producer,
					Inputs:        []Input{},
					Checks:        []Check{{ID: "go-test", Status: StatusPass, Summary: "go test ./..."}},
				})
			},
			wantStatus: StatusStale,
			wantReason: "artifact branch",
		},
		{
			name:   "stale head mismatch",
			gateID: "local-verification",
			prepare: func(t *testing.T, cfgDir string) {
				writeArtifactForTest(t, cfgDir, Artifact{
					SchemaVersion: "0.6.0",
					GateID:        "local-verification",
					Status:        StatusPass,
					RecordedAt:    time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC).Format(time.RFC3339),
					Subject:       Subject{HeadSHA: "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa", Branch: branch},
					Producer:      producer,
					Inputs:        []Input{},
					Checks:        []Check{{ID: "go-test", Status: StatusPass, Summary: "go test ./..."}},
				})
			},
			wantStatus: StatusStale,
			wantReason: "artifact head_sha",
		},
		{
			name:   "pass",
			gateID: "local-verification",
			prepare: func(t *testing.T, cfgDir string) {
				writeArtifactForTest(t, cfgDir, Artifact{
					SchemaVersion: "0.6.0",
					GateID:        "local-verification",
					Status:        StatusPass,
					RecordedAt:    time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC).Format(time.RFC3339),
					Subject:       subject,
					Producer:      producer,
					Inputs:        []Input{},
					Checks:        []Check{{ID: "go-test", Status: StatusPass, Summary: "go test ./..."}},
				})
			},
			wantStatus: StatusPass,
		},
		{
			name:   "fail",
			gateID: "local-verification",
			prepare: func(t *testing.T, cfgDir string) {
				writeArtifactForTest(t, cfgDir, Artifact{
					SchemaVersion: "0.6.0",
					GateID:        "local-verification",
					Status:        StatusFail,
					RecordedAt:    time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC).Format(time.RFC3339),
					Subject:       subject,
					Producer:      producer,
					Inputs:        []Input{},
					Checks:        []Check{{ID: "go-vet", Status: StatusFail, Summary: "go vet ./..."}},
				})
			},
			wantStatus: StatusFail,
		},
		{
			name:   "invalid contract",
			gateID: "local-coderabbit",
			prepare: func(t *testing.T, cfgDir string) {
				path, err := ArtifactPath(cfgDir, "local-coderabbit")
				if err != nil {
					t.Fatal(err)
				}
				writeFile(t, path, marshalArtifactForTest(t, Artifact{
					SchemaVersion: "0.6.0",
					GateID:        "local-coderabbit",
					Status:        StatusPass,
					RecordedAt:    time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC).Format(time.RFC3339),
					Subject:       subject,
					Producer:      Producer{Procedure: "change-inspect", Tool: "rgd gate record"},
					Inputs:        []Input{},
					Checks:        []Check{{ID: "wrong-check", Status: StatusPass, Summary: "missing CR proof"}},
				}))
			},
			wantStatus: StatusInvalid,
			wantReason: "local-coderabbit-cli",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			t.Parallel()
			cfgDir := t.TempDir()
			tc.prepare(t, cfgDir)

			res, err := Status(context.Background(), cfgDir, repo, tc.gateID)
			if err != nil {
				t.Fatal(err)
			}
			if res.Status != tc.wantStatus {
				t.Fatalf("status=%q want %q (%+v)", res.Status, tc.wantStatus, res)
			}
			if tc.wantReason != "" && !contains(res.Reason, tc.wantReason) {
				t.Fatalf("reason=%q want substring %q", res.Reason, tc.wantReason)
			}
		})
	}
}

func TestLoadSignals_injectsDerivedStatuses(t *testing.T) {
	t.Parallel()
	// Given: one valid artifact and one invalid artifact on disk
	repo := initGitRepo(t)
	cfgDir := t.TempDir()
	branch := currentBranchForTest(t, repo)
	head := currentHeadForTest(t, repo)
	writeArtifactForTest(t, cfgDir, Artifact{
		SchemaVersion: "0.6.0",
		GateID:        "local-verification",
		Status:        StatusPass,
		RecordedAt:    time.Date(2026, 4, 3, 12, 0, 0, 0, time.UTC).Format(time.RFC3339),
		Subject:       Subject{HeadSHA: head, Branch: branch},
		Producer:      Producer{Procedure: "implement", Tool: "rgd gate record"},
		Inputs:        []Input{},
		Checks:        []Check{{ID: "go-test", Status: StatusPass, Summary: "go test ./..."}},
	})
	path, err := ArtifactPath(cfgDir, "pr-readiness")
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, []byte(`{"bad":true}`))

	// When: LoadSignals scans local gate artifacts
	got, err := LoadSignals(context.Background(), cfgDir, repo)
	if err != nil {
		t.Fatal(err)
	}

	// Then: gate statuses are exposed under gates.<id>.status
	gates, ok := got["gates"].(map[string]any)
	if !ok {
		t.Fatalf("expected gates map, got %T", got["gates"])
	}
	local, ok := gates["local-verification"].(map[string]any)
	if !ok {
		t.Fatalf("expected local-verification map, got %T", gates["local-verification"])
	}
	if local["status"] != StatusPass {
		t.Fatalf("local-verification status=%v", local["status"])
	}
	pr, ok := gates["pr-readiness"].(map[string]any)
	if !ok {
		t.Fatalf("expected pr-readiness map, got %T", gates["pr-readiness"])
	}
	if pr["status"] != StatusInvalid {
		t.Fatalf("pr-readiness status=%v", pr["status"])
	}
}

func TestValidateGateID_rejectsSurroundingWhitespace(t *testing.T) {
	t.Parallel()
	if err := ValidateGateID(" local-verification "); err == nil {
		t.Fatal("expected error for whitespace-padded gate id")
	}
}

func initGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init")
	runGit(t, dir, "branch", "-M", "main")
	return dir
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
}

func currentHeadForTest(t *testing.T, dir string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

func currentBranchForTest(t *testing.T, dir string) string {
	t.Helper()
	out, err := exec.Command("git", "-C", dir, "symbolic-ref", "-q", "--short", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	return strings.TrimSpace(string(out))
}

func writeArtifactForTest(t *testing.T, cfgDir string, art Artifact) {
	t.Helper()
	path, err := ArtifactPath(cfgDir, art.GateID)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := validateArtifact(art); err != nil {
		t.Fatal(err)
	}
	if err := writeArtifactFile(path, art); err != nil {
		t.Fatal(err)
	}
}

func writeFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatal(err)
	}
}

func marshalArtifactForTest(t *testing.T, art Artifact) []byte {
	t.Helper()
	data, err := json.MarshalIndent(art, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	return append(data, '\n')
}

func contains(s, want string) bool {
	return strings.TrimSpace(s) != "" && strings.Contains(s, want)
}
