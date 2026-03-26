package githubapi

import (
	"context"
	"errors"
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

func TestRepoFromGH_success(t *testing.T) {
	// Given: stub repo view JSON query output
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
	// When: RepoFromGH runs
	owner, name, err := RepoFromGH(context.Background(), "")
	if err != nil {
		t.Fatal(err)
	}
	// Then: owner and repo name split
	if owner != "acme" || name != "widget" {
		t.Fatalf("owner/name: got %q %q", owner, name)
	}
}

func TestRepoFromGH_unexpectedNameWithOwner(t *testing.T) {
	// Given: malformed nameWithOwner string from gh
	old := runGHCommand
	t.Cleanup(func() { runGHCommand = old })
	runGHCommand = func(ctx context.Context, wd string, args []string) ([]byte, []byte, error) {
		return []byte("not-a-slash-separated-owner\n"), nil, nil
	}
	// When: RepoFromGH runs
	_, _, err := RepoFromGH(context.Background(), "")
	if err == nil {
		t.Fatal("expected error")
	}
	// Then: parse error
	if !strings.Contains(err.Error(), "unexpected nameWithOwner") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestRepoFromGH_errorIncludesStderr(t *testing.T) {
	// Given: gh failure with stderr
	old := runGHCommand
	t.Cleanup(func() { runGHCommand = old })
	runGHCommand = func(ctx context.Context, wd string, args []string) ([]byte, []byte, error) {
		return nil, []byte("permission denied\n"), errors.New("exit 1")
	}
	// When: RepoFromGH runs
	_, _, err := RepoFromGH(context.Background(), "")
	if err == nil {
		t.Fatal("expected error")
	}
	// Then: stderr surfaced in error
	if !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("expected stderr in error: %v", err)
	}
}
