package observe

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strconv"
	"strings"

	"github.com/k-shibuki/reinguard/internal/gitroot"
)

// GitProvider observes local git state via subprocess (ADR-0005).
type GitProvider struct{}

// NewGitProvider returns a git provider.
func NewGitProvider() *GitProvider {
	return &GitProvider{}
}

// GitProviderFactory builds a git provider from config options (ADR-0009).
// Options are reserved for future tuning; currently ignored.
func GitProviderFactory(opts map[string]any) (Provider, error) {
	_ = opts
	return NewGitProvider(), nil
}

// ID implements Provider.
func (*GitProvider) ID() string { return "git" }

// Collect implements Provider.
func (*GitProvider) Collect(ctx context.Context, opts Options) (Fragment, error) {
	wd := opts.WorkDir
	if wd == "" {
		return Fragment{}, fmt.Errorf("git provider: empty workdir")
	}
	if _, err := exec.LookPath("git"); err != nil {
		return Fragment{
			Degraded: true,
			Diagnostics: []Diagnostic{{
				Severity: "error",
				Message:  "git not found in PATH",
				Provider: "git",
				Code:     "git_missing",
			}},
		}, nil
	}
	return gitCollectFragment(ctx, wd, opts.DefaultBranch), nil
}

// gitCollectFragment gathers git signals and diagnostics for wd (git must be on PATH).
func gitCollectFragment(ctx context.Context, wd, defaultBranch string) Fragment {
	defaultBranch = strings.TrimSpace(defaultBranch)
	branch, detached, berr := gitroot.CurrentBranch(ctx, wd)
	porcelain, serr := gitRunOut(ctx, wd, "git", "status", "--porcelain")
	uncommitted := 0
	if serr == nil && strings.TrimSpace(porcelain) != "" {
		uncommitted = len(strings.Split(strings.TrimSpace(porcelain), "\n"))
	}
	stashCount, stashErr := gitStashCount(ctx, wd)
	ahead, behind, upstreamOK, upstreamErr := gitUpstreamAheadBehind(ctx, wd)
	staleRemote, staleDiag := gitStaleRemoteBranchesMerged(ctx, wd, defaultBranch)

	signals := map[string]any{}
	if berr == nil {
		signals["branch"] = branch
		signals["detached_head"] = detached
	}
	if serr == nil {
		signals["uncommitted_files"] = uncommitted
		signals["working_tree_clean"] = uncommitted == 0
	}
	if stashErr == nil {
		signals["stash_count"] = stashCount
	}
	if upstreamErr == nil {
		signals["ahead_of_upstream"] = ahead
		signals["behind_of_upstream"] = behind
		signals["has_upstream"] = upstreamOK
	}
	if len(staleDiag) == 0 {
		signals["stale_remote_branches_count"] = staleRemote
	}
	var diags []Diagnostic
	if berr != nil {
		diags = append(diags, Diagnostic{Severity: "warning", Message: berr.Error(), Provider: "git"})
	}
	if serr != nil {
		diags = append(diags, Diagnostic{Severity: "warning", Message: serr.Error(), Provider: "git"})
	}
	if stashErr != nil {
		diags = append(diags, Diagnostic{Severity: "warning", Message: stashErr.Error(), Provider: "git", Code: "stash_list_failed"})
	}
	if upstreamErr != nil {
		diags = append(diags, Diagnostic{Severity: "warning", Message: upstreamErr.Error(), Provider: "git", Code: "upstream_resolve_failed"})
	}
	diags = append(diags, staleDiag...)
	degraded := false
	for _, d := range diags {
		if d.Severity == "error" || d.Severity == "warning" {
			degraded = true
			break
		}
	}
	return Fragment{Signals: signals, Diagnostics: diags, Degraded: degraded}
}

func gitStashCount(ctx context.Context, wd string) (int, error) {
	out, err := gitRunOut(ctx, wd, "git", "stash", "list")
	if err != nil {
		return 0, err
	}
	if strings.TrimSpace(out) == "" {
		return 0, nil
	}
	return len(strings.Split(strings.TrimSpace(out), "\n")), nil
}

// gitUpstreamAheadBehind returns commits ahead/behind @{upstream}; upstreamOK false if none.
func gitUpstreamAheadBehind(ctx context.Context, wd string) (ahead, behind int, upstreamOK bool, err error) {
	if _, e := gitRunOut(ctx, wd, "git", "rev-parse", "-q", "--verify", "@{upstream}"); e != nil {
		return 0, 0, false, nil
	}
	aheadStr, err := gitRunOut(ctx, wd, "git", "rev-list", "--count", "@{upstream}..HEAD")
	if err != nil {
		return 0, 0, true, err
	}
	behindStr, err := gitRunOut(ctx, wd, "git", "rev-list", "--count", "HEAD..@{upstream}")
	if err != nil {
		return 0, 0, true, err
	}
	ahead = atoiNonneg(aheadStr)
	behind = atoiNonneg(behindStr)
	return ahead, behind, true, nil
}

// gitStaleRemoteBranchesMerged counts remote-tracking branches merged into origin/<DefaultBranch>,
// excluding the merge base ref itself and HEAD pointer lines, when mergeBaseRef exists.
func gitStaleRemoteBranchesMerged(ctx context.Context, wd, defaultBranch string) (int, []Diagnostic) {
	var diags []Diagnostic
	if strings.TrimSpace(defaultBranch) == "" {
		diags = append(diags, Diagnostic{
			Severity: "warning",
			Message:  "default_branch unset; stale_remote_branches_count is 0",
			Provider: "git",
			Code:     "stale_branches_skipped",
		})
		return 0, diags
	}
	mergeRef := "origin/" + defaultBranch
	if _, err := gitRunOut(ctx, wd, "git", "rev-parse", "-q", "--verify", mergeRef); err != nil {
		// No such remote ref — common in fresh `git init` without fetch.
		return 0, diags
	}
	out, err := gitRunOut(ctx, wd, "git", "branch", "-r", "--merged", mergeRef)
	if err != nil {
		diags = append(diags, Diagnostic{
			Severity: "warning",
			Message:  err.Error(),
			Provider: "git",
			Code:     "stale_remote_branches_failed",
		})
		return 0, diags
	}
	n := 0
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		// Skip symbolic refs (e.g. "origin/HEAD -> origin/main", "HEAD -> origin/main").
		if line == "" || strings.Contains(line, " -> ") {
			continue
		}
		// Exclude the merge base ref itself (every branch is trivially merged into itself).
		if line == mergeRef {
			continue
		}
		n++
	}
	return n, diags
}

func atoiNonneg(s string) int {
	v, err := strconv.Atoi(strings.TrimSpace(s))
	if err != nil || v < 0 {
		return 0
	}
	return v
}

func gitRunOut(ctx context.Context, wd string, name string, args ...string) (string, error) {
	var buf bytes.Buffer
	cmd := exec.CommandContext(ctx, name, args...)
	cmd.Dir = wd
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("%s %v: %w: %s", name, args, err, strings.TrimSpace(buf.String()))
	}
	return strings.TrimSpace(buf.String()), nil
}
