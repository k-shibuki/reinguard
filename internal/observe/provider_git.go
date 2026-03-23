package observe

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// GitProvider observes local git state via subprocess (ADR-0005).
type GitProvider struct{}

// NewGitProvider returns a git provider.
func NewGitProvider() *GitProvider {
	return &GitProvider{}
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

	branch, detached, berr := gitHead(ctx, wd)
	porcelain, serr := gitRunOut(ctx, wd, "git", "status", "--porcelain")
	uncommitted := 0
	if serr == nil {
		if strings.TrimSpace(porcelain) != "" {
			lines := strings.Split(strings.TrimSpace(porcelain), "\n")
			uncommitted = len(lines)
		}
	}

	stashCount, stashErr := gitStashCount(ctx, wd)
	ahead, behind, upstreamOK, upstreamErr := gitUpstreamAheadBehind(ctx, wd)
	staleRemote, staleDiag := gitStaleRemoteBranchesMerged(ctx, wd, opts.DefaultBranch)

	signals := map[string]any{
		"branch":                      branch,
		"detached_head":               detached,
		"uncommitted_files":           uncommitted,
		"working_tree_clean":          uncommitted == 0,
		"stash_count":                 stashCount,
		"ahead_of_upstream":           ahead,
		"behind_of_upstream":          behind,
		"has_upstream":                upstreamOK,
		"stale_remote_branches_count": staleRemote,
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
	return Fragment{Signals: signals, Diagnostics: diags, Degraded: degraded}, nil
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

// gitStaleRemoteBranchesMerged counts remote-tracking branches merged into origin/<DefaultBranch>.
// Semantics: lines from `git branch -r --merged <mergeBaseRef>` minus HEAD pointer lines, when mergeBaseRef exists.
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
		if line == "" || strings.HasPrefix(line, "HEAD ->") {
			continue
		}
		n++
	}
	return n, diags
}

func atoiNonneg(s string) int {
	s = strings.TrimSpace(s)
	if s == "" {
		return 0
	}
	var v int
	for _, c := range s {
		if c < '0' || c > '9' {
			return 0
		}
		v = v*10 + int(c-'0')
	}
	return v
}

func gitHead(ctx context.Context, wd string) (branch string, detached bool, err error) {
	out, err := gitRunOut(ctx, wd, "git", "symbolic-ref", "-q", "--short", "HEAD")
	if err == nil {
		return strings.TrimSpace(out), false, nil
	}
	if _, err2 := gitRunOut(ctx, wd, "git", "rev-parse", "--verify", "HEAD"); err2 == nil {
		return "", true, nil
	}
	return "", false, err
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
