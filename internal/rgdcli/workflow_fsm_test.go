package rgdcli

import (
	"bytes"
	"encoding/json"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func reinguardConfigDir(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	src := filepath.Join(root, ".reinguard")
	dst := filepath.Join(t.TempDir(), ".reinguard")
	if err := copyTree(t, src, dst); err != nil {
		t.Fatalf("copy .reinguard: %v", err)
	}
	// Isolate FSM scenario tests from developer-local pr-readiness artifacts (pass would force ready_for_pr).
	_ = os.Remove(filepath.Join(dst, "runtime", "gates", "pr-readiness.json"))
	return dst
}

func copyTree(t *testing.T, src, dst string) error {
	t.Helper()
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		out := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(out, 0o755)
		}
		b, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(out, b, info.Mode().Perm())
	})
}

// workflowFSMScenarioFixtures pairs observation JSON with expected state_id and route_id (ADR-0013).
var workflowFSMScenarioFixtures = []struct {
	name        string
	observation string
	wantStateID string
	wantRouteID string
}{
	{
		name: "working_no_pr_missing_pr_flag",
		observation: `{
  "signals": {
    "git": {"detached_head": false, "working_tree_clean": true}
  },
  "degraded": false
}`,
		wantStateID: "working_no_pr",
		wantRouteID: "user-implement",
	},
	{
		name: "working_no_pr_explicit_false",
		observation: `{
  "signals": {
    "git": {"detached_head": false},
    "github": {
      "pull_requests": {"pr_exists_for_branch": false}
    }
  },
  "degraded": false
}`,
		wantStateID: "working_no_pr",
		wantRouteID: "user-implement",
	},
	{
		name: "waiting_ci_pending",
		observation: `{
  "signals": {
    "git": {"detached_head": false, "working_tree_clean": true},
    "github": {
      "pull_requests": {
        "pr_exists_for_branch": true,
        "merge_state_status": "unstable"
      },
      "ci": {"ci_status": "pending"},
      "reviews": {
        "review_threads_unresolved": 0,
        "pagination_incomplete": false,
        "review_decisions_changes_requested": 0,
        "review_decisions_truncated": false,
        "bot_reviewer_status": []
      }
    }
  },
  "degraded": false
}`,
		wantStateID: "waiting_ci",
		wantRouteID: "user-wait-ci",
	},
	{
		name: "unresolved_threads",
		observation: `{
  "signals": {
    "git": {"detached_head": false, "working_tree_clean": true},
    "github": {
      "pull_requests": {
        "pr_exists_for_branch": true,
        "merge_state_status": "unstable"
      },
      "ci": {"ci_status": "pending"},
      "reviews": {
        "review_threads_unresolved": 1,
        "review_decisions_changes_requested": 0,
        "bot_reviewer_status": []
      }
    }
  },
  "degraded": false
}`,
		wantStateID: "unresolved_threads",
		wantRouteID: "user-address-review",
	},
	{
		name: "changes_requested",
		observation: `{
  "signals": {
    "git": {"detached_head": false, "working_tree_clean": true},
    "github": {
      "pull_requests": {"pr_exists_for_branch": true, "merge_state_status": "dirty"},
      "ci": {"ci_status": "success"},
      "reviews": {
        "review_threads_unresolved": 0,
        "review_decisions_changes_requested": 1,
        "bot_reviewer_status": []
      }
    }
  },
  "degraded": false
}`,
		wantStateID: "changes_requested",
		wantRouteID: "user-address-review",
	},
	{
		name: "merge_ready",
		observation: `{
  "signals": {
    "git": {"detached_head": false, "working_tree_clean": true},
    "github": {
      "pull_requests": {
        "pr_exists_for_branch": true,
        "merge_state_status": "clean"
      },
      "ci": {"ci_status": "success"},
      "reviews": {
        "review_threads_unresolved": 0,
        "pagination_incomplete": false,
        "review_decisions_changes_requested": 0,
        "review_decisions_truncated": false,
        "bot_reviewer_status": []
      }
    }
  },
  "degraded": false
}`,
		wantStateID: "merge_ready",
		wantRouteID: "user-merge",
	},
	{
		name: "unresolved_threads_beats_bot_pending",
		observation: `{
  "signals": {
    "git": {"detached_head": false, "working_tree_clean": true},
    "github": {
      "pull_requests": {"pr_exists_for_branch": true, "merge_state_status": "unstable"},
      "ci": {"ci_status": "pending"},
      "reviews": {
        "review_threads_unresolved": 1,
        "review_decisions_changes_requested": 0,
        "bot_reviewer_status": [],
        "bot_review_diagnostics": {
          "bot_review_failed": false,
          "bot_review_completed": false,
          "bot_review_pending": true,
          "bot_review_terminal": false
        }
      }
    }
  },
  "degraded": false
}`,
		wantStateID: "unresolved_threads",
		wantRouteID: "user-address-review",
	},
	{
		name: "bot_rate_limited",
		observation: `{
  "signals": {
    "git": {"detached_head": false},
    "github": {
      "pull_requests": {"pr_exists_for_branch": true},
      "reviews": {
        "review_threads_unresolved": 0,
        "review_decisions_changes_requested": 0,
        "bot_reviewer_status": [
          {"login": "bot", "required": true, "status": "rate_limited", "contains_rate_limit": true}
        ]
      }
    }
  },
  "degraded": false
}`,
		wantStateID: "waiting_bot_rate_limited",
		wantRouteID: "user-wait-bot-quota",
	},
	{
		name: "bot_review_paused",
		observation: `{
  "signals": {
    "git": {"detached_head": false},
    "github": {
      "pull_requests": {"pr_exists_for_branch": true},
      "reviews": {
        "review_threads_unresolved": 0,
        "review_decisions_changes_requested": 0,
        "bot_reviewer_status": [
          {"login": "bot", "required": true, "status": "review_paused", "contains_review_paused": true}
        ]
      }
    }
  },
  "degraded": false
}`,
		wantStateID: "waiting_bot_paused",
		wantRouteID: "user-wait-bot-paused",
	},
	{
		name: "bot_review_failed",
		observation: `{
  "signals": {
    "git": {"detached_head": false},
    "github": {
      "pull_requests": {"pr_exists_for_branch": true},
      "reviews": {
        "review_threads_unresolved": 0,
        "review_decisions_changes_requested": 0,
        "bot_reviewer_status": [],
        "bot_review_diagnostics": {
          "bot_review_failed": true,
          "bot_review_completed": false,
          "bot_review_pending": false,
          "bot_review_terminal": true
        }
      }
    }
  },
  "degraded": false
}`,
		wantStateID: "waiting_bot_failed",
		wantRouteID: "user-wait-bot-failed",
	},
	{
		name: "bot_reviewing",
		observation: `{
  "signals": {
    "git": {"detached_head": false},
    "github": {
      "pull_requests": {"pr_exists_for_branch": true},
      "reviews": {
        "review_threads_unresolved": 0,
        "review_decisions_changes_requested": 0,
        "bot_reviewer_status": [],
        "bot_review_diagnostics": {
          "bot_review_failed": false,
          "bot_review_completed": false,
          "bot_review_pending": true,
          "bot_review_terminal": false
        }
      }
    }
  },
  "degraded": false
}`,
		wantStateID: "waiting_bot_run",
		wantRouteID: "user-wait-bot-run",
	},
}

func TestRunStateEval_workflowFSM_observationScenarios(t *testing.T) {
	// Given: workflow observation JSON with various GitHub/git signal combinations
	// When: rgd state eval is invoked with --config-dir pointing to .reinguard
	// Then: the resolved state_id matches the expected FSM state for each scenario
	t.Parallel()
	cfgDir := reinguardConfigDir(t)

	for _, tt := range workflowFSMScenarioFixtures {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			obsDir := t.TempDir()
			p := filepath.Join(obsDir, "obs.json")
			if err := os.WriteFile(p, []byte(tt.observation), 0o644); err != nil {
				t.Fatal(err)
			}
			var buf bytes.Buffer
			app := NewApp("test")
			app.Writer = &buf
			if err := app.Run([]string{
				"rgd", "state", "eval",
				"--config-dir", cfgDir,
				"--observation-file", p,
			}); err != nil {
				t.Fatal(err)
			}
			var out map[string]any
			if err := json.Unmarshal(buf.Bytes(), &out); err != nil {
				t.Fatalf("json: %v raw=%s", err, buf.String())
			}
			if out["kind"] != "resolved" {
				t.Fatalf("kind=%v want resolved: %s", out["kind"], buf.String())
			}
			if out["state_id"] != tt.wantStateID {
				t.Fatalf("state_id=%v want %q", out["state_id"], tt.wantStateID)
			}
		})
	}
}

func TestRunRouteSelect_workflowFSM_resolvesRoute(t *testing.T) {
	// Given: observation JSON that resolves to a known workflow state_id
	// When: rgd route select runs with state eval output as --state-file
	// Then: route_id matches ADR-0013 primary route for that state
	t.Parallel()
	cfgDir := reinguardConfigDir(t)

	for _, tt := range workflowFSMScenarioFixtures {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			obsDir := t.TempDir()
			obsPath := filepath.Join(obsDir, "obs.json")
			writeFile(t, obsPath, []byte(tt.observation))
			stateDir := t.TempDir()
			var sbuf bytes.Buffer
			app := NewApp("t1")
			app.Writer = &sbuf
			if err := app.Run([]string{
				"rgd", "state", "eval",
				"--config-dir", cfgDir,
				"--observation-file", obsPath,
			}); err != nil {
				t.Fatal(err)
			}
			writeFile(t, filepath.Join(stateDir, "state.json"), sbuf.Bytes())

			var rbuf bytes.Buffer
			app2 := NewApp("t2")
			app2.Writer = &rbuf
			if err := app2.Run([]string{
				"rgd", "route", "select",
				"--config-dir", cfgDir,
				"--observation-file", obsPath,
				"--state-file", filepath.Join(stateDir, "state.json"),
			}); err != nil {
				t.Fatal(err)
			}
			var routeOut map[string]any
			if err := json.Unmarshal(rbuf.Bytes(), &routeOut); err != nil {
				t.Fatalf("json: %v", err)
			}
			if routeOut["kind"] != "resolved" {
				t.Fatalf("kind=%v body=%s", routeOut["kind"], rbuf.String())
			}
			if routeOut["route_id"] != tt.wantRouteID {
				t.Fatalf("route_id=%v want %q", routeOut["route_id"], tt.wantRouteID)
			}
		})
	}
}

func TestRunStateEval_workflowFSM_readyForPRFromGate(t *testing.T) {
	t.Parallel()
	// Given/When/Then: actual workflow rules see a passing pr-readiness gate as ready_for_pr
	repo := initGitRepoForGateCLI(t)
	cfgDir := writeWorkflowFSMConfig(t)
	obsPath := filepath.Join(t.TempDir(), "obs.json")
	writeFile(t, obsPath, []byte(`{
  "signals": {
    "git": {"detached_head": false},
    "github": {
      "pull_requests": {"pr_exists_for_branch": false}
    }
  },
  "degraded": false
}`))

	var recordBuf bytes.Buffer
	recordApp := NewApp("record")
	recordApp.Writer = &recordBuf
	if err := recordApp.Run([]string{
		"rgd", "gate", "record",
		"--config-dir", cfgDir,
		"--cwd", repo,
		"--status", "pass",
		"pr-readiness",
	}); err != nil {
		t.Fatal(err)
	}

	var stateBuf bytes.Buffer
	stateApp := NewApp("state")
	stateApp.Writer = &stateBuf
	if err := stateApp.Run([]string{
		"rgd", "state", "eval",
		"--config-dir", cfgDir,
		"--cwd", repo,
		"--observation-file", obsPath,
	}); err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(stateBuf.Bytes(), &out); err != nil {
		t.Fatalf("json: %v raw=%s", err, stateBuf.String())
	}
	if out["state_id"] != "ready_for_pr" {
		t.Fatalf("state_id=%v want ready_for_pr", out["state_id"])
	}

	statePath := filepath.Join(t.TempDir(), "state.json")
	writeFile(t, statePath, stateBuf.Bytes())
	var routeBuf bytes.Buffer
	routeApp := NewApp("route")
	routeApp.Writer = &routeBuf
	if err := routeApp.Run([]string{
		"rgd", "route", "select",
		"--config-dir", cfgDir,
		"--cwd", repo,
		"--observation-file", obsPath,
		"--state-file", statePath,
	}); err != nil {
		t.Fatal(err)
	}
	var routeOut map[string]any
	if err := json.Unmarshal(routeBuf.Bytes(), &routeOut); err != nil {
		t.Fatalf("json: %v raw=%s", err, routeBuf.String())
	}
	if routeOut["route_id"] != "user-implement" {
		t.Fatalf("route_id=%v want user-implement", routeOut["route_id"])
	}
}

func TestRunStateEval_workflowFSM_stalePrReadinessFallsBackToWorkingNoPR(t *testing.T) {
	t.Parallel()
	// Given/When/Then: a stale pr-readiness artifact must not keep the branch in ready_for_pr
	repo := initGitRepoForGateCLI(t)
	cfgDir := writeWorkflowFSMConfig(t)
	obsPath := filepath.Join(t.TempDir(), "obs.json")
	writeFile(t, obsPath, []byte(`{
  "signals": {
    "git": {"detached_head": false},
    "github": {
      "pull_requests": {"pr_exists_for_branch": false}
    }
  },
  "degraded": false
}`))

	recordApp := NewApp("record")
	recordApp.Writer = io.Discard
	if err := recordApp.Run([]string{
		"rgd", "gate", "record",
		"--config-dir", cfgDir,
		"--cwd", repo,
		"--status", "pass",
		"pr-readiness",
	}); err != nil {
		t.Fatal(err)
	}
	runGitForGateCLI(t, repo, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "advance")

	var stateBuf bytes.Buffer
	stateApp := NewApp("state")
	stateApp.Writer = &stateBuf
	if err := stateApp.Run([]string{
		"rgd", "state", "eval",
		"--config-dir", cfgDir,
		"--cwd", repo,
		"--observation-file", obsPath,
	}); err != nil {
		t.Fatal(err)
	}
	var out map[string]any
	if err := json.Unmarshal(stateBuf.Bytes(), &out); err != nil {
		t.Fatalf("json: %v raw=%s", err, stateBuf.String())
	}
	if out["state_id"] != "working_no_pr" {
		t.Fatalf("state_id=%v want working_no_pr", out["state_id"])
	}
}

func writeWorkflowFSMConfig(t *testing.T) string {
	t.Helper()
	cfgDir := t.TempDir()
	writeFile(t, filepath.Join(cfgDir, "reinguard.yaml"), []byte(testFixtureReinguardRoot))
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	repoReinguard := filepath.Join(root, ".reinguard")
	copyWorkflowFSMFile(t, filepath.Join(repoReinguard, "control", "states", "workflow.yaml"), filepath.Join(cfgDir, "control", "states", "workflow.yaml"))
	copyWorkflowFSMFile(t, filepath.Join(repoReinguard, "control", "routes", "workflow.yaml"), filepath.Join(cfgDir, "control", "routes", "workflow.yaml"))
	return cfgDir
}

func copyWorkflowFSMFile(t *testing.T, src, dst string) {
	t.Helper()
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, dst, data)
}
