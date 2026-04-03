package scripttest

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupLocalReviewRepo(t *testing.T) string {
	t.Helper()

	repo := t.TempDir()
	cmd := exec.Command("git", "init", "-q")
	cmd.Dir = repo
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git init: %v\n%s", err, out)
	}
	if err := os.WriteFile(filepath.Join(repo, ".coderabbit.yaml"), []byte("reviews:\n  auto_review:\n    enabled: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return repo
}

func TestCheckLocalReviewScript_RetryUsesLatestRateLimitLine(t *testing.T) {
	t.Parallel()

	script := scriptPath(t, "check-local-review.sh")
	// Given: a temporary git repo with stubbed CodeRabbit CLI and sleep commands.
	repo := setupLocalReviewRepo(t)

	stubDir := t.TempDir()
	logFile := filepath.Join(stubDir, "coderabbit.log")
	countFile := filepath.Join(stubDir, "coderabbit-count.txt")
	sleepFile := filepath.Join(stubDir, "sleep.log")

	writeExecutable(t, stubDir, "coderabbit", `#!/usr/bin/env bash
set -euo pipefail
log_file="${TEST_LOG_FILE:?}"
count_file="${TEST_COUNT_FILE:?}"
subcmd="$1"
shift
case "$subcmd" in
  auth)
    echo "logged in"
    ;;
  review)
    count=0
    if [[ -f "$count_file" ]]; then
      count=$(cat "$count_file")
    fi
    count=$((count + 1))
    printf '%s\n' "$count" >"$count_file"
    echo "review attempt $count: $*" >>"$log_file"
    if [[ $count -eq 1 ]]; then
      cat <<'EOF'
noise: 99 minutes old
[2026-04-03T00:00:00Z] ERROR: Rate limit exceeded, please try after 9 minutes and 9 seconds
intermediate note
[2026-04-03T00:00:01Z] ERROR: Rate limit exceeded, please try after 1 seconds
EOF
      exit 1
    fi
    cat <<'EOF'
Review completed: 0 findings
EOF
    ;;
  *)
    echo "unexpected subcommand: $subcmd" >&2
    exit 1
    ;;
esac
`)
	writeExecutable(t, stubDir, "sleep", `#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$1" >>"${TEST_SLEEP_FILE:?}"
`)

	env := []string{
		"PATH=" + stubDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		"TEST_LOG_FILE=" + logFile,
		"TEST_COUNT_FILE=" + countFile,
		"TEST_SLEEP_FILE=" + sleepFile,
		"RATE_LIMIT_RETRY_BUFFER_SEC=30",
	}

	// When: the local review script runs with automatic retry enabled.
	out, err := runBashScript(t, repo, script, env, "--base", "main", "--retry-on-rate-limit")
	if err != nil {
		t.Fatalf("check-local-review: %v\n%s", err, out)
	}

	// Then: it uses only the latest rate-limit line and sleeps for 1s + 30s buffer.
	sleepLog, err := os.ReadFile(sleepFile)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(sleepLog)); got != "31" {
		t.Fatalf("sleep seconds = %q, want 31", got)
	}
	logData, err := os.ReadFile(logFile)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(logData), "review attempt") != 2 {
		t.Fatalf("expected two review attempts, got log:\n%s", logData)
	}
	if !strings.Contains(out, "CodeRabbit local review completed.") {
		t.Fatalf("expected completion message, got:\n%s", out)
	}
}

func TestCheckLocalReviewScript_UnparseableLatestRateLimitFailsClosed(t *testing.T) {
	t.Parallel()

	script := scriptPath(t, "check-local-review.sh")
	// Given: a temporary git repo with a stubbed CodeRabbit CLI that emits an unparseable rate-limit line.
	repo := setupLocalReviewRepo(t)

	stubDir := t.TempDir()
	sleepFile := filepath.Join(stubDir, "sleep.log")
	writeExecutable(t, stubDir, "coderabbit", `#!/usr/bin/env bash
set -euo pipefail
subcmd="$1"
shift
case "$subcmd" in
  auth)
    echo "logged in"
    ;;
  review)
    cat <<'EOF'
[2026-04-03T00:00:00Z] ERROR: Rate limit exceeded, please try again later
EOF
    exit 1
    ;;
  *)
    exit 1
    ;;
esac
`)
	writeExecutable(t, stubDir, "sleep", `#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$1" >>"${TEST_SLEEP_FILE:?}"
`)

	env := []string{
		"PATH=" + stubDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		"RATE_LIMIT_RETRY_BUFFER_SEC=30",
		"TEST_SLEEP_FILE=" + sleepFile,
	}

	// When: the local review script runs with automatic retry enabled.
	out, err := runBashScript(t, repo, script, env, "--base", "main", "--retry-on-rate-limit")

	// Then: it stops with a parse failure tied to the latest rate-limit line.
	if err == nil {
		t.Fatalf("expected failure, got success:\n%s", out)
	}
	if !strings.Contains(out, "could not be parsed from the latest rate-limit line") {
		t.Fatalf("expected parse failure message, got:\n%s", out)
	}
	if _, err := os.Stat(sleepFile); err == nil {
		sleepLog, readErr := os.ReadFile(sleepFile)
		if readErr != nil {
			t.Fatal(readErr)
		}
		t.Fatalf("expected fail-closed behavior without sleep, got sleep log:\n%s", sleepLog)
	} else if !os.IsNotExist(err) {
		t.Fatalf("unexpected error checking sleep file: %v", err)
	}
}
