package githubapi

import (
	"context"
	"errors"
	"strings"
	"testing"
)

func TestReplyToPullRequestThread(t *testing.T) {
	tests := []struct {
		stub    func(ctx context.Context, wd string, stdin []byte, args []string) ([]byte, []byte, error)
		name    string
		wd      string
		wantErr string
		input   PullRequestThreadReplyInput
	}{
		{
			name: "given valid input when replying then request is posted",
			wd:   "/tmp/work",
			input: PullRequestThreadReplyInput{
				PRNumber:  17,
				Body:      "Fixed. Updated the scoped observation path.",
				InReplyTo: 123,
				CommitSHA: "0123456789abcdef0123456789abcdef01234567",
				Path:      " internal/rgdcli/rgdcli.go ",
				Line:      42,
			},
			stub: func(ctx context.Context, wd string, stdin []byte, args []string) ([]byte, []byte, error) {
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
			},
		},
		{
			name: "given short sha when replying then validation fails",
			input: PullRequestThreadReplyInput{
				PRNumber:  1,
				Body:      "x",
				InReplyTo: 2,
				CommitSHA: "deadbeef",
				Path:      "a.go",
				Line:      1,
			},
			wantErr: "40-character",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: an optional stubbed gh transport
			if tt.stub != nil {
				old := runGHCommandWithInput
				t.Cleanup(func() { runGHCommandWithInput = old })
				runGHCommandWithInput = tt.stub
			}

			// When: the helper posts a threaded reply
			err := ReplyToPullRequestThread(context.Background(), tt.wd, tt.input)

			// Then: the error state matches the expected transport outcome
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("got %v, want error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}

func TestResolveReviewThread(t *testing.T) {
	tests := []struct {
		stub    func(ctx context.Context, wd string, args []string) ([]byte, []byte, error)
		name    string
		wd      string
		thread  string
		wantErr string
	}{
		{
			name:   "given resolved response when resolving then success",
			wd:     "/tmp/work",
			thread: "THREAD_node_id",
			stub: func(ctx context.Context, wd string, args []string) ([]byte, []byte, error) {
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
			},
		},
		{
			name:    "given gh stderr when resolving then error preserved",
			thread:  "THREAD_node_id",
			wantErr: "denied",
			stub: func(ctx context.Context, wd string, args []string) ([]byte, []byte, error) {
				return nil, []byte("denied"), errors.New("exit 1")
			},
		},
		{
			name:    "given graphql errors when resolving then fail closed",
			thread:  "THREAD_node_id",
			wantErr: "permission denied",
			stub: func(ctx context.Context, wd string, args []string) ([]byte, []byte, error) {
				return []byte(`{"errors":[{"message":"permission denied"}]}`), nil, nil
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Given: a stubbed gh GraphQL transport
			old := runGHCommand
			t.Cleanup(func() { runGHCommand = old })
			runGHCommand = tt.stub

			// When: the helper resolves the requested thread
			err := ResolveReviewThread(context.Background(), tt.wd, tt.thread)

			// Then: the parsed result matches the expected success or failure mode
			if tt.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
					t.Fatalf("got %v, want error containing %q", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
		})
	}
}
