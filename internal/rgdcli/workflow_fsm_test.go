package rgdcli

import (
	"bytes"
	"encoding/json"
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
	return filepath.Join(root, ".reinguard")
}

func TestRunStateEval_workflowFSM_observationScenarios(t *testing.T) {
	t.Parallel()
	cfgDir := reinguardConfigDir(t)

	tests := []struct {
		name        string
		observation string
		wantStateID string
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
		},
		{
			name: "pr_open",
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
        "tracked_reviewer_status": []
      }
    }
  },
  "degraded": false
}`,
			wantStateID: "pr_open",
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
        "tracked_reviewer_status": []
      }
    }
  },
  "degraded": false
}`,
			wantStateID: "changes_requested",
		},
		{
			name: "ready_to_merge",
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
        "review_decisions_changes_requested": 0,
        "tracked_reviewer_status": []
      }
    }
  },
  "degraded": false
}`,
			wantStateID: "ready_to_merge",
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
        "tracked_reviewer_status": [
          {"login": "bot", "contains_rate_limit": true, "contains_review_paused": false}
        ]
      }
    }
  },
  "degraded": false
}`,
			wantStateID: "bot_rate_limited",
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
        "tracked_reviewer_status": [
          {"login": "bot", "contains_rate_limit": false, "contains_review_paused": true}
        ]
      }
    }
  },
  "degraded": false
}`,
			wantStateID: "bot_review_paused",
		},
	}
	for _, tt := range tests {
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
	t.Parallel()
	cfgDir := reinguardConfigDir(t)
	obsDir := t.TempDir()
	obs := `{
  "signals": {
    "git": {"detached_head": false},
    "github": {"pull_requests": {"pr_exists_for_branch": false}}
  },
  "degraded": false
}`
	writeFile(t, filepath.Join(obsDir, "obs.json"), []byte(obs))
	stateDir := t.TempDir()
	var sbuf bytes.Buffer
	app := NewApp("t1")
	app.Writer = &sbuf
	if err := app.Run([]string{
		"rgd", "state", "eval",
		"--config-dir", cfgDir,
		"--observation-file", filepath.Join(obsDir, "obs.json"),
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
		"--observation-file", filepath.Join(obsDir, "obs.json"),
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
	if routeOut["route_id"] != "cursor-implement" {
		t.Fatalf("route_id=%v want cursor-implement", routeOut["route_id"])
	}
}
