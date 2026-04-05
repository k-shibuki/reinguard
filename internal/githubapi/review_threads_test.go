package githubapi

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestReplyToPullRequestThread_success(t *testing.T) {
	// Given: a stubbed gh API transport for threaded replies
	old := runGHCommandWithInput
	t.Cleanup(func() { runGHCommandWithInput = old })
	runGHCommandWithInput = func(ctx context.Context, wd string, stdin []byte, args []string) ([]byte, []byte, error) {
		if wd != "/tmp/work" {
			t.Fatalf("unexpected wd: %q", wd)
		}
		if !strings.Contains(string(stdin), `"in_reply_to":123`) {
			t.Fatalf("stdin: %s", stdin)
		}
		if !strings.Contains(string(stdin), `"commit_id":"0123456789abcdef0123456789abcdef01234567"`) {
			t.Fatalf("stdin: %s", stdin)
		}
		if !strings.Contains(string(stdin), `"path":"internal/rgdcli/rgdcli.go"`) {
			t.Fatalf("stdin: %s", stdin)
		}
		if len(args) < 2 || args[0] != "api" || args[1] != "repos/{owner}/{repo}/pulls/17/comments" {
			t.Fatalf("unexpected args: %v", args)
		}
		return []byte(`{}`), nil, nil
	}
	// When: the helper posts a valid threaded reply
	err := ReplyToPullRequestThread(context.Background(), "/tmp/work", PullRequestThreadReplyInput{
		PRNumber:  17,
		Body:      "Fixed. Updated the scoped observation path.",
		InReplyTo: 123,
		CommitSHA: "0123456789abcdef0123456789abcdef01234567",
		Path:      " internal/rgdcli/rgdcli.go ",
		Line:      42,
	})
	// Then: the transport succeeds and trims payload fields as needed
	if err != nil {
		t.Fatal(err)
	}
}

func TestReplyToPullRequestThread_rejectsShortSHA(t *testing.T) {
	// Given: reply input with a short SHA
	// When: the helper validates the input
	err := ReplyToPullRequestThread(context.Background(), "", PullRequestThreadReplyInput{
		PRNumber:  1,
		Body:      "x",
		InReplyTo: 2,
		CommitSHA: "deadbeef",
		Path:      "a.go",
		Line:      1,
	})
	// Then: the helper rejects the short SHA before any transport call
	if err == nil || !strings.Contains(err.Error(), "40-character") {
		t.Fatalf("got %v", err)
	}
}

func TestResolveReviewThread_success(t *testing.T) {
	// Given: a stubbed GraphQL response that confirms resolution
	old := runGHCommand
	t.Cleanup(func() { runGHCommand = old })
	runGHCommand = func(ctx context.Context, wd string, args []string) ([]byte, []byte, error) {
		if wd != "/tmp/work" {
			t.Fatalf("unexpected wd: %q", wd)
		}
		if len(args) < 4 || args[0] != "api" || args[1] != "graphql" {
			t.Fatalf("unexpected args: %v", args)
		}
		if !strings.Contains(args[3], "THREAD_node_id") {
			t.Fatalf("query: %v", args)
		}
		return []byte(`{"data":{"resolveReviewThread":{"thread":{"isResolved":true}}}}`), nil, nil
	}
	// When: the helper resolves the thread
	if err := ResolveReviewThread(context.Background(), "/tmp/work", "THREAD_node_id"); err != nil {
		t.Fatal(err)
	}
}

func TestResolveReviewThread_propagatesGHError(t *testing.T) {
	// Given: gh exits non-zero before GraphQL response parsing
	old := runGHCommand
	t.Cleanup(func() { runGHCommand = old })
	runGHCommand = func(ctx context.Context, wd string, args []string) ([]byte, []byte, error) {
		return nil, []byte("denied"), errors.New("exit 1")
	}
	// When: the helper attempts to resolve the thread
	err := ResolveReviewThread(context.Background(), "", "THREAD_node_id")
	// Then: stderr is preserved in the returned error
	if err == nil || !strings.Contains(err.Error(), "denied") {
		t.Fatalf("got %v", err)
	}
}

func TestResolveReviewThread_rejectsGraphQLErrors(t *testing.T) {
	// Given: gh succeeds but GraphQL returns an errors array
	old := runGHCommand
	t.Cleanup(func() { runGHCommand = old })
	runGHCommand = func(ctx context.Context, wd string, args []string) ([]byte, []byte, error) {
		return []byte(`{"errors":[{"message":"permission denied"}]}`), nil, nil
	}
	// When: the helper parses the mutation response
	err := ResolveReviewThread(context.Background(), "", "THREAD_node_id")
	// Then: GraphQL errors fail closed
	if err == nil || !strings.Contains(err.Error(), "permission denied") {
		t.Fatalf("got %v", err)
	}
}
