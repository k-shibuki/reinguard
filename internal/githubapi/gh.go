package githubapi

import (
	"bytes"
	"context"
	"fmt"
	"net/url"
	"os/exec"
	"strings"
)

// runGHCommand runs the GitHub CLI subprocess. Tests replace it for hermetic runs.
var runGHCommand = runGHCommandImpl

func runGHCommandImpl(ctx context.Context, wd string, args []string) (stdout, stderr []byte, err error) {
	cmd := exec.CommandContext(ctx, "gh", args...)
	if wd != "" {
		cmd.Dir = wd
	}
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return outBuf.Bytes(), errBuf.Bytes(), err
}

// TokenFromGH runs `gh auth token` (ADR-0006).
func TokenFromGH(ctx context.Context, wd string) (string, error) {
	out, stderr, err := runGHCommand(ctx, wd, []string{"auth", "token"})
	if err != nil {
		return "", fmt.Errorf("gh auth token: %w (stderr: %s)", err, strings.TrimSpace(string(stderr)))
	}
	return strings.TrimSpace(string(out)), nil
}

// runGitCommand runs the git subprocess. Tests replace it for hermetic runs.
var runGitCommand = runGitCommandImpl

func runGitCommandImpl(ctx context.Context, wd string, args []string) (stdout, stderr []byte, err error) {
	cmd := exec.CommandContext(ctx, "git", args...)
	if wd != "" {
		cmd.Dir = wd
	}
	var outBuf, errBuf bytes.Buffer
	cmd.Stdout = &outBuf
	cmd.Stderr = &errBuf
	err = cmd.Run()
	return outBuf.Bytes(), errBuf.Bytes(), err
}

func splitNameWithOwner(raw string) (owner, name string, err error) {
	s := strings.TrimSpace(raw)
	parts := strings.Split(s, "/")
	if len(parts) != 2 {
		return "", "", fmt.Errorf("unexpected nameWithOwner %q", s)
	}
	owner = strings.TrimSpace(parts[0])
	name = strings.TrimSpace(parts[1])
	if owner == "" || name == "" {
		return "", "", fmt.Errorf("unexpected nameWithOwner %q", s)
	}
	return owner, name, nil
}

func splitGitHubRemotePath(path string) (owner, name string, err error) {
	path = strings.Trim(strings.TrimSpace(path), "/")
	path = strings.TrimSuffix(path, ".git")
	return splitNameWithOwner(path)
}

func repoFromGitRemote(ctx context.Context, wd string) (owner, name string, err error) {
	out, stderr, err := runGitCommand(ctx, wd, []string{"config", "--get", "remote.origin.url"})
	if err != nil {
		return "", "", fmt.Errorf("git remote get origin url: %w (stderr: %s)", err, strings.TrimSpace(string(stderr)))
	}
	raw := strings.TrimSpace(string(out))
	switch {
	case strings.HasPrefix(raw, "git@github.com:"):
		return splitGitHubRemotePath(strings.TrimPrefix(raw, "git@github.com:"))
	case strings.HasPrefix(raw, "ssh://git@github.com/"):
		return splitGitHubRemotePath(strings.TrimPrefix(raw, "ssh://git@github.com/"))
	}
	u, err := url.Parse(raw)
	if err != nil {
		return "", "", fmt.Errorf("parse remote.origin.url %q: %w", raw, err)
	}
	if !strings.EqualFold(u.Hostname(), "github.com") {
		return "", "", fmt.Errorf("remote.origin.url host %q is not github.com", u.Hostname())
	}
	return splitGitHubRemotePath(u.Path)
}

func repoFromGHRepoView(ctx context.Context, wd string) (owner, name string, err error) {
	args := []string{"repo", "view", "--json", "nameWithOwner", "-q", ".nameWithOwner"}
	out, stderr, err := runGHCommand(ctx, wd, args)
	if err != nil {
		return "", "", fmt.Errorf("gh repo view: %w (stderr: %s)", err, strings.TrimSpace(string(stderr)))
	}
	return splitNameWithOwner(string(out))
}

// GitHubRepoIdentitySource indicates how owner/name were resolved.
type GitHubRepoIdentitySource string

const (
	// RepoIdentitySourceLocalGit means owner/name came from parsing remote.origin.url for github.com.
	RepoIdentitySourceLocalGit GitHubRepoIdentitySource = "local_git"
	// RepoIdentitySourceGHRepoView means owner/name came from `gh repo view` (fallback).
	RepoIdentitySourceGHRepoView GitHubRepoIdentitySource = "gh_repo_view"
)

// GitHubRepoIdentity is the resolved GitHub repository identity for the current workdir.
type GitHubRepoIdentity struct {
	Owner  string
	Name   string
	Source GitHubRepoIdentitySource
}

// ResolveGitHubRepoIdentityFromWorkDir resolves owner/name from local Git remote first (local-first),
// then falls back to `gh repo view` when origin is missing or not a standard github.com remote.
func ResolveGitHubRepoIdentityFromWorkDir(ctx context.Context, wd string) (GitHubRepoIdentity, error) {
	owner, name, err := repoFromGitRemote(ctx, wd)
	if err == nil {
		return GitHubRepoIdentity{Owner: owner, Name: name, Source: RepoIdentitySourceLocalGit}, nil
	}
	gitErr := err
	owner, name, err = repoFromGHRepoView(ctx, wd)
	if err == nil {
		return GitHubRepoIdentity{Owner: owner, Name: name, Source: RepoIdentitySourceGHRepoView}, nil
	}
	return GitHubRepoIdentity{}, fmt.Errorf("resolve repo identity: local git: %v; gh repo view: %w", gitErr, err)
}

// RepoFromGH resolves repository identity using [ResolveGitHubRepoIdentityFromWorkDir].
// Deprecated: prefer [ResolveGitHubRepoIdentityFromWorkDir] for source visibility.
func RepoFromGH(ctx context.Context, wd string) (owner, name string, err error) {
	id, err := ResolveGitHubRepoIdentityFromWorkDir(ctx, wd)
	if err != nil {
		return "", "", err
	}
	return id.Owner, id.Name, nil
}
