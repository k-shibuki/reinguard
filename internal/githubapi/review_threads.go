package githubapi

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
)

// PullRequestThreadReplyInput is the transport payload for one threaded PR review reply.
type PullRequestThreadReplyInput struct {
	Body      string
	CommitSHA string
	Path      string
	PRNumber  int
	InReplyTo int
	Line      int
}

// ReplyToPullRequestThread posts a threaded reply on one pull-request review comment.
func ReplyToPullRequestThread(ctx context.Context, wd string, in PullRequestThreadReplyInput) error {
	if in.PRNumber <= 0 {
		return fmt.Errorf("pull request number must be greater than 0")
	}
	if in.InReplyTo <= 0 {
		return fmt.Errorf("in_reply_to must be greater than 0")
	}
	in.Body = strings.TrimSpace(in.Body)
	if in.Body == "" {
		return fmt.Errorf("reply body must be non-empty")
	}
	sha, err := validateFullSHA(in.CommitSHA)
	if err != nil {
		return err
	}
	in.CommitSHA = sha
	in.Path = strings.TrimSpace(in.Path)
	if in.Path == "" {
		return fmt.Errorf("path must be non-empty")
	}
	if in.Line <= 0 {
		return fmt.Errorf("line must be greater than 0")
	}
	payload, err := json.Marshal(map[string]any{
		"body":        in.Body,
		"in_reply_to": in.InReplyTo,
		"commit_id":   in.CommitSHA,
		"path":        in.Path,
		"line":        in.Line,
	})
	if err != nil {
		return err
	}
	_, stderr, err := runGHCommandWithInput(ctx, wd, payload, []string{
		"api",
		fmt.Sprintf("repos/{owner}/{repo}/pulls/%d/comments", in.PRNumber),
		"-X", "POST",
		"--input", "-",
	})
	if err != nil {
		return fmt.Errorf("gh api reply thread: %w (stderr: %s)", err, strings.TrimSpace(string(stderr)))
	}
	return nil
}

// ResolveReviewThread resolves one GraphQL review thread after consensus.
func ResolveReviewThread(ctx context.Context, wd, threadID string) error {
	threadID = strings.TrimSpace(threadID)
	if threadID == "" {
		return fmt.Errorf("thread id must be non-empty")
	}
	query := fmt.Sprintf(`mutation { resolveReviewThread(input: {threadId: %q}) { thread { isResolved } } }`, threadID)
	stdout, stderr, err := runGHCommand(ctx, wd, []string{"api", "graphql", "-f", "query=" + query})
	if err != nil {
		return fmt.Errorf("gh api resolve review thread: %w (stderr: %s)", err, strings.TrimSpace(string(stderr)))
	}
	var resp struct {
		Data struct {
			ResolveReviewThread struct {
				Thread *struct {
					IsResolved bool `json:"isResolved"`
				} `json:"thread"`
			} `json:"resolveReviewThread"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := json.Unmarshal(stdout, &resp); err != nil {
		return fmt.Errorf("parse resolve response: %w", err)
	}
	if len(resp.Errors) > 0 {
		return fmt.Errorf("graphql error: %s", resp.Errors[0].Message)
	}
	if resp.Data.ResolveReviewThread.Thread == nil {
		return fmt.Errorf("thread %s not found in response", threadID)
	}
	if !resp.Data.ResolveReviewThread.Thread.IsResolved {
		return fmt.Errorf("thread %s was not resolved", threadID)
	}
	return nil
}
