package scripttest

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/k-shibuki/reinguard/pkg/schema"
)

func requireExitCode(t *testing.T, err error, want int, out string) {
	t.Helper()
	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		t.Fatalf("expected *exec.ExitError, got %T: %v\n%s", err, err, out)
	}
	if got := exitErr.ExitCode(); got != want {
		t.Fatalf("exit code = %d, want %d\n%s", got, want, out)
	}
}

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

func writeLocalReviewReinguardConfig(t *testing.T, repo, waitSeconds string) {
	t.Helper()
	cfgDir := filepath.Join(repo, ".reinguard")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	cfg := fmt.Sprintf(`schema_version: %q
default_branch: main
workflow:
  local_ai_review:
    coderabbit:
      unknown_quota_wait_seconds: %s
providers: []
`, schema.CurrentSchemaVersion, waitSeconds)
	if err := os.WriteFile(filepath.Join(cfgDir, "reinguard.yaml"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}
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
    echo "Authentication: logged in"
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
    echo "Authentication: logged in"
    ;;
  review)
    cat <<'EOF'
[2026-04-03T00:00:00Z] ERROR: Rate limit exceeded, please try after cooldown reset
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

	// Then: it stops with a parse failure tied to the latest cooldown instruction line.
	if err == nil {
		t.Fatalf("expected failure, got success:\n%s", out)
	}
	requireExitCode(t, err, 2, out)
	if !strings.Contains(out, "duration could not be parsed from the latest cooldown instruction line") {
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

func TestCheckLocalReviewScript_RateLimitFooterDoesNotSupplyCooldown(t *testing.T) {
	t.Parallel()

	script := scriptPath(t, "check-local-review.sh")
	repo := setupLocalReviewRepo(t)

	stubDir := t.TempDir()
	sleepFile := filepath.Join(stubDir, "sleep.log")
	writeExecutable(t, stubDir, "coderabbit", `#!/usr/bin/env bash
set -euo pipefail
subcmd="$1"
shift
case "$subcmd" in
  auth)
    echo "Authentication: logged in"
    ;;
  review)
    cat <<'EOF'
[2026-04-03T00:00:00Z] ERROR: Rate limit exceeded, please try after cooldown reset
Job finished in 5 seconds
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

	out, err := runBashScript(t, repo, script, env, "--base", "main", "--retry-on-rate-limit")
	if err == nil {
		t.Fatalf("expected failure, got success:\n%s", out)
	}
	requireExitCode(t, err, 2, out)
	if !strings.Contains(out, "duration could not be parsed from the latest cooldown instruction line") {
		t.Fatalf("expected parse failure message, got:\n%s", out)
	}
	if _, err := os.Stat(sleepFile); err == nil {
		sleepLog, readErr := os.ReadFile(sleepFile)
		if readErr != nil {
			t.Fatal(readErr)
		}
		t.Fatalf("expected fail-closed without sleep; footer must not supply cooldown, got:\n%s", sleepLog)
	} else if !os.IsNotExist(err) {
		t.Fatalf("unexpected error checking sleep file: %v", err)
	}
}

func TestCheckLocalReviewScript_UnknownQuotaGuidanceUsesConfiguredFallback(t *testing.T) {
	t.Parallel()

	script := scriptPath(t, "check-local-review.sh")
	// Given: CodeRabbit emits usage-based/hourly-cap guidance without retry-after, and repo config supplies fallback wait.
	repo := setupLocalReviewRepo(t)
	writeLocalReviewReinguardConfig(t, repo, "1860")

	stubDir := t.TempDir()
	sleepFile := filepath.Join(stubDir, "sleep.log")
	countFile := filepath.Join(stubDir, "coderabbit-count.txt")
	writeExecutable(t, stubDir, "coderabbit", `#!/usr/bin/env bash
set -euo pipefail
subcmd="$1"; shift
case "$subcmd" in
  auth) echo "Authentication: logged in" ;;
  review)
    count=0; [[ -f "${TEST_COUNT_FILE:?}" ]] && count=$(cat "${TEST_COUNT_FILE:?}")
    count=$((count + 1)); printf '%s\n' "$count" >"${TEST_COUNT_FILE:?}"
    if [[ $count -eq 1 ]]; then
      echo "ERROR: To keep reviews running without waiting, you can enable usage-based add-on for your organization. This allows additional reviews beyond the hourly cap."
      exit 1
    fi
    echo "Review completed: 0 findings"
    ;;
  *) exit 1 ;;
esac
`)
	writeExecutable(t, stubDir, "sleep", `#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$1" >>"${TEST_SLEEP_FILE:?}"
`)

	env := []string{
		"PATH=" + stubDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		"TEST_SLEEP_FILE=" + sleepFile,
		"TEST_COUNT_FILE=" + countFile,
	}

	// When: automatic retry is enabled.
	out, err := runBashScript(t, repo, script, env, "--base", "main", "--retry-on-rate-limit")
	if err != nil {
		t.Fatalf("check-local-review: %v\n%s", err, out)
	}

	// Then: the configured fallback wait is used before the one retry.
	sleepLog, err := os.ReadFile(sleepFile)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(sleepLog)); got != "1860" {
		t.Fatalf("sleep seconds = %q, want 1860", got)
	}
	reviewCount, err := os.ReadFile(countFile)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(reviewCount)); got != "2" {
		t.Fatalf("review attempts = %q, want 2", got)
	}
	if !strings.Contains(out, "CodeRabbit local review completed.") {
		t.Fatalf("expected completion message, got:\n%s", out)
	}
}

func TestCheckLocalReviewScript_UnknownQuotaGuidanceRequiresConfiguredFallback(t *testing.T) {
	t.Parallel()

	script := scriptPath(t, "check-local-review.sh")
	// Given: CodeRabbit emits usage-based/hourly-cap guidance, but repo config has no fallback wait.
	repo := setupLocalReviewRepo(t)

	stubDir := t.TempDir()
	sleepFile := filepath.Join(stubDir, "sleep.log")
	writeExecutable(t, stubDir, "coderabbit", `#!/usr/bin/env bash
set -euo pipefail
subcmd="$1"; shift
case "$subcmd" in
  auth) echo "Authentication: logged in" ;;
  review)
    echo "ERROR: To keep reviews running without waiting, you can enable usage-based add-on for your organization. This allows additional reviews beyond the hourly cap."
    exit 1
    ;;
  *) exit 1 ;;
esac
`)
	writeExecutable(t, stubDir, "sleep", `#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$1" >>"${TEST_SLEEP_FILE:?}"
`)

	env := []string{
		"PATH=" + stubDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		"TEST_SLEEP_FILE=" + sleepFile,
	}

	// When: automatic retry is enabled.
	out, err := runBashScript(t, repo, script, env, "--base", "main", "--retry-on-rate-limit")

	// Then: the script fails closed instead of inventing a wait.
	if err == nil {
		t.Fatalf("expected failure, got success:\n%s", out)
	}
	requireExitCode(t, err, 2, out)
	if !strings.Contains(out, "unknown_quota_wait_seconds is missing or invalid") {
		t.Fatalf("expected missing fallback diagnostic, got:\n%s", out)
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

func TestCheckLocalReviewScript_UnknownQuotaGuidanceRejectsInvalidFallback(t *testing.T) {
	t.Parallel()

	script := scriptPath(t, "check-local-review.sh")
	// Given: CodeRabbit emits usage-based/hourly-cap guidance, but repo config has an invalid fallback wait.
	repo := setupLocalReviewRepo(t)
	writeLocalReviewReinguardConfig(t, repo, "not-a-number")

	stubDir := t.TempDir()
	sleepFile := filepath.Join(stubDir, "sleep.log")
	writeExecutable(t, stubDir, "coderabbit", `#!/usr/bin/env bash
set -euo pipefail
subcmd="$1"; shift
case "$subcmd" in
  auth) echo "Authentication: logged in" ;;
  review)
    echo "ERROR: To keep reviews running without waiting, you can enable usage-based add-on for your organization. This allows additional reviews beyond the hourly cap."
    exit 1
    ;;
  *) exit 1 ;;
esac
`)
	writeExecutable(t, stubDir, "sleep", `#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$1" >>"${TEST_SLEEP_FILE:?}"
`)

	env := []string{
		"PATH=" + stubDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		"TEST_SLEEP_FILE=" + sleepFile,
	}

	// When: automatic retry is enabled.
	out, err := runBashScript(t, repo, script, env, "--base", "main", "--retry-on-rate-limit")

	// Then: the script fails closed instead of sleeping with an invalid fallback.
	if err == nil {
		t.Fatalf("expected failure, got success:\n%s", out)
	}
	requireExitCode(t, err, 2, out)
	if !strings.Contains(out, "unknown_quota_wait_seconds is missing or invalid") {
		t.Fatalf("expected invalid fallback diagnostic, got:\n%s", out)
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

func TestCheckLocalReviewScript_ExplicitCooldownPrecedesUnknownQuotaFallback(t *testing.T) {
	t.Parallel()

	script := scriptPath(t, "check-local-review.sh")
	// Given: CodeRabbit output includes billing guidance and a parseable cooldown line.
	repo := setupLocalReviewRepo(t)
	writeLocalReviewReinguardConfig(t, repo, "1860")

	stubDir := t.TempDir()
	sleepFile := filepath.Join(stubDir, "sleep.log")
	countFile := filepath.Join(stubDir, "coderabbit-count.txt")
	writeExecutable(t, stubDir, "coderabbit", `#!/usr/bin/env bash
set -euo pipefail
subcmd="$1"; shift
case "$subcmd" in
  auth) echo "Authentication: logged in" ;;
  review)
    count=0; [[ -f "${TEST_COUNT_FILE:?}" ]] && count=$(cat "${TEST_COUNT_FILE:?}")
    count=$((count + 1)); printf '%s\n' "$count" >"${TEST_COUNT_FILE:?}"
    if [[ $count -eq 1 ]]; then
      echo "usage-based add-on notice"
      echo "Rate limit exceeded, retry in 1 seconds"
      exit 1
    fi
    echo "Review completed: 0 findings"
    ;;
  *) exit 1 ;;
esac
`)
	writeExecutable(t, stubDir, "sleep", `#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$1" >>"${TEST_SLEEP_FILE:?}"
`)

	env := []string{
		"PATH=" + stubDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		"TEST_SLEEP_FILE=" + sleepFile,
		"TEST_COUNT_FILE=" + countFile,
		"RATE_LIMIT_RETRY_BUFFER_SEC=30",
	}

	// When: automatic retry is enabled.
	out, err := runBashScript(t, repo, script, env, "--base", "main", "--retry-on-rate-limit")
	if err != nil {
		t.Fatalf("check-local-review: %v\n%s", err, out)
	}

	// Then: explicit cooldown plus buffer wins over the configured unknown-quota fallback.
	sleepLog, err := os.ReadFile(sleepFile)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(sleepLog)); got != "31" {
		t.Fatalf("sleep seconds = %q, want 31", got)
	}
	reviewCount, err := os.ReadFile(countFile)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(reviewCount)); got != "2" {
		t.Fatalf("review attempts = %q, want 2", got)
	}
}

func TestCheckLocalReviewScript_UnknownQuotaGuidanceSecondHitFails(t *testing.T) {
	t.Parallel()

	script := scriptPath(t, "check-local-review.sh")
	// Given: CodeRabbit emits usage-based/hourly-cap guidance on both attempts.
	repo := setupLocalReviewRepo(t)
	writeLocalReviewReinguardConfig(t, repo, "1860")

	stubDir := t.TempDir()
	sleepFile := filepath.Join(stubDir, "sleep.log")
	countFile := filepath.Join(stubDir, "coderabbit-count.txt")
	writeExecutable(t, stubDir, "coderabbit", `#!/usr/bin/env bash
set -euo pipefail
subcmd="$1"; shift
case "$subcmd" in
  auth) echo "Authentication: logged in" ;;
  review)
    count=0; [[ -f "${TEST_COUNT_FILE:?}" ]] && count=$(cat "${TEST_COUNT_FILE:?}")
    count=$((count + 1)); printf '%s\n' "$count" >"${TEST_COUNT_FILE:?}"
    echo "ERROR: To keep reviews running without waiting, you can enable usage-based add-on for your organization. This allows additional reviews beyond the hourly cap."
    exit 1
    ;;
  *) exit 1 ;;
esac
`)
	writeExecutable(t, stubDir, "sleep", `#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$1" >>"${TEST_SLEEP_FILE:?}"
`)

	env := []string{
		"PATH=" + stubDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		"TEST_SLEEP_FILE=" + sleepFile,
		"TEST_COUNT_FILE=" + countFile,
	}

	// When: automatic retry is enabled.
	out, err := runBashScript(t, repo, script, env, "--base", "main", "--retry-on-rate-limit")

	// Then: the script retries once and then fails with a quota-specific diagnostic.
	if err == nil {
		t.Fatalf("expected failure, got success:\n%s", out)
	}
	requireExitCode(t, err, 2, out)
	if !strings.Contains(out, "usage-based/hourly-cap quota guidance again after automatic retry") {
		t.Fatalf("expected second quota hit diagnostic, got:\n%s", out)
	}
	sleepLog, err := os.ReadFile(sleepFile)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(sleepLog)); got != "1860" {
		t.Fatalf("sleep seconds = %q, want one 1860 wait", got)
	}
	reviewCount, err := os.ReadFile(countFile)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(reviewCount)); got != "2" {
		t.Fatalf("review attempts = %q, want 2", got)
	}
}

func TestCheckLocalReviewScript_RetryZeroSecondsUsesBufferOnly(t *testing.T) {
	t.Parallel()

	script := scriptPath(t, "check-local-review.sh")
	repo := setupLocalReviewRepo(t)

	stubDir := t.TempDir()
	sleepFile := filepath.Join(stubDir, "sleep.log")
	countFile := filepath.Join(stubDir, "coderabbit-count.txt")

	writeExecutable(t, stubDir, "coderabbit", `#!/usr/bin/env bash
set -euo pipefail
subcmd="$1"; shift
case "$subcmd" in
  auth) echo "Authentication: logged in" ;;
  review)
    count=0; [[ -f "${TEST_COUNT_FILE:?}" ]] && count=$(cat "${TEST_COUNT_FILE:?}")
    count=$((count + 1)); printf '%s\n' "$count" >"${TEST_COUNT_FILE:?}"
    if [[ $count -eq 1 ]]; then
      echo "[ts] ERROR: Rate limit exceeded, retry in 0 seconds"
      exit 1
    fi
    echo "Review completed: 0 findings"
    ;;
  *) exit 1 ;;
esac
`)
	writeExecutable(t, stubDir, "sleep", `#!/usr/bin/env bash
set -euo pipefail
printf '%s\n' "$1" >>"${TEST_SLEEP_FILE:?}"
`)

	env := []string{
		"PATH=" + stubDir + string(os.PathListSeparator) + os.Getenv("PATH"),
		"TEST_SLEEP_FILE=" + sleepFile,
		"TEST_COUNT_FILE=" + countFile,
		"RATE_LIMIT_RETRY_BUFFER_SEC=30",
	}
	out, err := runBashScript(t, repo, script, env, "--base", "main", "--retry-on-rate-limit")
	if err != nil {
		t.Fatalf("check-local-review: %v\n%s", err, out)
	}
	reviewCount, err := os.ReadFile(countFile)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(reviewCount)); got != "2" {
		t.Fatalf("review attempts = %q, want 2", got)
	}
	sleepLog, err := os.ReadFile(sleepFile)
	if err != nil {
		t.Fatal(err)
	}
	if got := strings.TrimSpace(string(sleepLog)); got != "30" {
		t.Fatalf("sleep seconds = %q, want 30", got)
	}
	if !strings.Contains(out, "CodeRabbit local review completed.") {
		t.Fatalf("expected completion message, got:\n%s", out)
	}
}

func TestCheckLocalReviewScript_AuthStatusNotCurrentlyLoggedInFailsClosed(t *testing.T) {
	t.Parallel()

	script := scriptPath(t, "check-local-review.sh")
	repo := setupLocalReviewRepo(t)

	stubDir := t.TempDir()
	writeExecutable(t, stubDir, "coderabbit", `#!/usr/bin/env bash
set -euo pipefail
subcmd="$1"; shift
case "$subcmd" in
  auth)
    echo "You are not currently logged in."
    ;;
  *)
    exit 1
    ;;
esac
`)

	env := []string{
		"PATH=" + stubDir + string(os.PathListSeparator) + os.Getenv("PATH"),
	}
	out, err := runBashScript(t, repo, script, env, "--base", "main")
	if err == nil {
		t.Fatalf("expected failure, got success:\n%s", out)
	}
	requireExitCode(t, err, 2, out)
	if !strings.Contains(out, "CodeRabbit CLI is not authenticated") {
		t.Fatalf("expected unauthenticated error, got:\n%s", out)
	}
}

func TestCheckLocalReviewScript_TryAgainInWithoutRateLimitPhraseRetries(t *testing.T) {
	t.Parallel()

	script := scriptPath(t, "check-local-review.sh")
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
    echo "Authentication: logged in"
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
Your review has hit the rate limit for this repository. Please try again in 1 minute and 1 second.
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

	out, err := runBashScript(t, repo, script, env, "--base", "main", "--retry-on-rate-limit")
	if err != nil {
		t.Fatalf("check-local-review: %v\n%s", err, out)
	}

	sleepLog, err := os.ReadFile(sleepFile)
	if err != nil {
		t.Fatal(err)
	}
	// 1 minute + 1 second = 61s cooldown + 30s buffer
	if got := strings.TrimSpace(string(sleepLog)); got != "91" {
		t.Fatalf("sleep seconds = %q, want 91", got)
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

func TestCheckLocalReviewScript_GeneralFailureWithoutCooldownHintFailsClosed(t *testing.T) {
	t.Parallel()

	script := scriptPath(t, "check-local-review.sh")
	repo := setupLocalReviewRepo(t)

	stubDir := t.TempDir()
	sleepFile := filepath.Join(stubDir, "sleep.log")
	writeExecutable(t, stubDir, "coderabbit", `#!/usr/bin/env bash
set -euo pipefail
subcmd="$1"
shift
case "$subcmd" in
  auth)
    echo "Authentication: logged in"
    ;;
  review)
    cat <<'EOF'
ERROR: Something went wrong while contacting the review service.
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

	out, err := runBashScript(t, repo, script, env, "--base", "main", "--retry-on-rate-limit")
	if err == nil {
		t.Fatalf("expected failure, got success:\n%s", out)
	}
	requireExitCode(t, err, 2, out)
	if !strings.Contains(out, "CodeRabbit local review failed. Resolve the CLI error") {
		t.Fatalf("expected general failure message, got:\n%s", out)
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
