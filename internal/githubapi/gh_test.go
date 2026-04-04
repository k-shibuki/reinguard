package githubapi

import (
	"context"
	"errors"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestTokenFromGH_success(t *testing.T) {
	// Given: stub gh auth token success
	old := runGHCommand
	t.Cleanup(func() { runGHCommand = old })
	runGHCommand = func(ctx context.Context, wd string, args []string) ([]byte, []byte, error) {
		if len(args) != 2 || args[0] != "auth" || args[1] != "token" {
			t.Fatalf("unexpected args: %v", args)
		}
		if wd != "/tmp/work" {
			t.Fatalf("unexpected wd: %q", wd)
		}
		return []byte("gho_test_token\n"), nil, nil
	}
	// When: TokenFromGH runs
	got, err := TokenFromGH(context.Background(), "/tmp/work")
	if err != nil {
		t.Fatal(err)
	}
	// Then: trimmed token string
	if got != "gho_test_token" {
		t.Fatalf("token: got %q", got)
	}
}

func TestTokenFromGH_errorIncludesStderr(t *testing.T) {
	// Given: stub gh failure with stderr
	old := runGHCommand
	t.Cleanup(func() { runGHCommand = old })
	runGHCommand = func(ctx context.Context, wd string, args []string) ([]byte, []byte, error) {
		return nil, []byte("not logged in\n"), errors.New("exit 1")
	}
	// When: TokenFromGH runs
	_, err := TokenFromGH(context.Background(), "")
	if err == nil {
		t.Fatal("expected error")
	}
	// Then: error includes stderr text
	if !strings.Contains(err.Error(), "not logged in") {
		t.Fatalf("expected stderr in error: %v", err)
	}
}

func TestResolveGitHubRepoIdentity_ghRepoViewWhenNoOrigin(t *testing.T) {
	// Given: stub repo view JSON query output (no origin in repo — gh path used)
	dir := t.TempDir()
	runGitCmd(t, dir, "init")
	old := runGHCommand
	t.Cleanup(func() { runGHCommand = old })
	runGHCommand = func(ctx context.Context, wd string, args []string) ([]byte, []byte, error) {
		want := []string{"repo", "view", "--json", "nameWithOwner", "-q", ".nameWithOwner"}
		if len(args) != len(want) {
			t.Fatalf("args len: got %v", args)
		}
		for i := range want {
			if args[i] != want[i] {
				t.Fatalf("args[%d]: got %q want %q", i, args[i], want[i])
			}
		}
		return []byte("acme/widget\n"), nil, nil
	}
	// When
	id, err := ResolveGitHubRepoIdentityFromWorkDir(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	// Then
	if id.Owner != "acme" || id.Name != "widget" {
		t.Fatalf("owner/name: got %q %q", id.Owner, id.Name)
	}
	if id.Source != RepoIdentitySourceGHRepoView {
		t.Fatalf("source: got %q", id.Source)
	}
}

func TestResolveGitHubRepoIdentity_unexpectedNameWithOwner(t *testing.T) {
	dir := t.TempDir()
	runGitCmd(t, dir, "init")
	old := runGHCommand
	t.Cleanup(func() { runGHCommand = old })
	runGHCommand = func(ctx context.Context, wd string, args []string) ([]byte, []byte, error) {
		return []byte("not-a-slash-separated-owner\n"), nil, nil
	}
	_, err := ResolveGitHubRepoIdentityFromWorkDir(context.Background(), dir)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "unexpected nameWithOwner") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestResolveGitHubRepoIdentity_ghErrorIncludesStderr(t *testing.T) {
	dir := t.TempDir()
	runGitCmd(t, dir, "init")
	old := runGHCommand
	t.Cleanup(func() { runGHCommand = old })
	runGHCommand = func(ctx context.Context, wd string, args []string) ([]byte, []byte, error) {
		return nil, []byte("permission denied\n"), errors.New("exit 1")
	}
	_, err := ResolveGitHubRepoIdentityFromWorkDir(context.Background(), dir)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("expected stderr in error: %v", err)
	}
}

func TestResolveGitHubRepoIdentity_runGitCommandStubbed(t *testing.T) {
	dir := t.TempDir()
	oldGit := runGitCommand
	oldGH := runGHCommand
	t.Cleanup(func() {
		runGitCommand = oldGit
		runGHCommand = oldGH
	})
	runGitCommand = func(ctx context.Context, wd string, args []string) ([]byte, []byte, error) {
		want := []string{"config", "--get", "remote.origin.url"}
		if len(args) != len(want) {
			t.Fatalf("args len: got %v", args)
		}
		for i := range want {
			if args[i] != want[i] {
				t.Fatalf("args[%d]: got %q want %q", i, args[i], want[i])
			}
		}
		if wd != dir {
			t.Fatalf("unexpected wd: %q", wd)
		}
		return []byte("https://github.com/acme/widget.git\n"), nil, nil
	}
	runGHCommand = func(ctx context.Context, wd string, args []string) ([]byte, []byte, error) {
		t.Fatal("gh should not be called when git remote resolves")
		return nil, nil, nil
	}

	id, err := ResolveGitHubRepoIdentityFromWorkDir(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if id.Owner != "acme" || id.Name != "widget" {
		t.Fatalf("owner/name: got %q %q", id.Owner, id.Name)
	}
	if id.Source != RepoIdentitySourceLocalGit {
		t.Fatalf("source: got %q", id.Source)
	}
}

func TestResolveGitHubRepoIdentity_localGitPreferredNoGhCall(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("git remote fixture uses Unix-oriented helpers elsewhere in this package")
	}
	var ghCalls int
	old := runGHCommand
	t.Cleanup(func() { runGHCommand = old })
	runGHCommand = func(ctx context.Context, wd string, args []string) ([]byte, []byte, error) {
		ghCalls++
		return nil, nil, errors.New("gh should not be needed when origin resolves")
	}

	dir := t.TempDir()
	runGitCmd(t, dir, "init")
	runGitCmd(t, dir, "remote", "add", "origin", "git@github.com:acme/widget.git")

	id, err := ResolveGitHubRepoIdentityFromWorkDir(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if ghCalls != 0 {
		t.Fatalf("expected gh not called, got %d calls", ghCalls)
	}
	if id.Owner != "acme" || id.Name != "widget" {
		t.Fatalf("owner/name: got %q %q", id.Owner, id.Name)
	}
	if id.Source != RepoIdentitySourceLocalGit {
		t.Fatalf("source: got %q", id.Source)
	}
}

func TestResolveGitHubRepoIdentity_bothFailMessage(t *testing.T) {
	old := runGHCommand
	t.Cleanup(func() { runGHCommand = old })
	runGHCommand = func(ctx context.Context, wd string, args []string) ([]byte, []byte, error) {
		return nil, []byte("permission denied\n"), errors.New("exit 1")
	}

	dir := t.TempDir()
	runGitCmd(t, dir, "init")

	_, err := ResolveGitHubRepoIdentityFromWorkDir(context.Background(), dir)
	if err == nil {
		t.Fatal("expected error")
	}
	if !strings.Contains(err.Error(), "permission denied") || !strings.Contains(err.Error(), "local git:") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRepoFromGH_matchesResolve(t *testing.T) {
	dir := t.TempDir()
	runGitCmd(t, dir, "init")
	old := runGHCommand
	t.Cleanup(func() { runGHCommand = old })
	runGHCommand = func(ctx context.Context, wd string, args []string) ([]byte, []byte, error) {
		return []byte("acme/widget\n"), nil, nil
	}
	o, n, err := RepoFromGH(context.Background(), dir)
	if err != nil {
		t.Fatal(err)
	}
	if o != "acme" || n != "widget" {
		t.Fatalf("got %q %q", o, n)
	}
}

func TestSplitGitHubRemotePath_supportsHTTPSPath(t *testing.T) {
	owner, name, err := splitGitHubRemotePath("/acme/widget.git")
	if err != nil {
		t.Fatal(err)
	}
	if owner != "acme" || name != "widget" {
		t.Fatalf("owner/name: got %q %q", owner, name)
	}
}

func TestSplitNameWithOwner_trimsPartWhitespace(t *testing.T) {
	owner, name, err := splitNameWithOwner("acme / widget")
	if err != nil {
		t.Fatal(err)
	}
	if owner != "acme" || name != "widget" {
		t.Fatalf("owner/name: got %q %q", owner, name)
	}
}

func runGitCmd(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v in %s: %v %s", args, filepath.Base(dir), err, string(out))
	}
}
