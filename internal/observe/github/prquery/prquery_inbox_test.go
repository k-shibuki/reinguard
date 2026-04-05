package prquery

import (
	"testing"
)

func TestBuildReviewInboxEntry_fallbackOriginalAnchors(t *testing.T) {
	t.Parallel()
	// Given: GitHub may omit current line/commit on outdated threads while original* remains.
	ln := 40
	start := 38
	origCommit := "deadbeefdeadbeefdeadbeefdeadbeefdeadbeef"
	thread := reviewThreadNode{
		ID:         "THREAD_FALLBACK",
		IsResolved: false,
		IsOutdated: true,
		Comments: &reviewThreadCommentsConn{
			Nodes: []reviewThreadCommentNode{{
				DatabaseID:        9001,
				Body:              "fix scope",
				Path:              "internal/x.go",
				Line:              nil,
				OriginalLine:      &ln,
				StartLine:         nil,
				OriginalStartLine: &start,
				Author: &struct {
					Login string `json:"login"`
				}{Login: "coderabbitai[bot]"},
				Commit: nil,
				OriginalCommit: &struct {
					Oid string `json:"oid"`
				}{Oid: origCommit},
			}},
		},
	}
	// When
	entry := buildReviewInboxEntry(thread)
	// Then: canonical anchors match original* for reply-thread use
	if entry == nil {
		t.Fatal("expected entry")
	}
	if got := entry["line"]; got != ln {
		t.Fatalf("line: got %v want %d", got, ln)
	}
	if got := entry["start_line"]; got != start {
		t.Fatalf("start_line: got %v want %d", got, start)
	}
	if got := entry["commit_sha"]; got != origCommit {
		t.Fatalf("commit_sha: got %v want %q", got, origCommit)
	}
	if _, ok := entry["original_line"]; !ok {
		t.Fatal("expected original_line preserved")
	}
	if _, ok := entry["original_commit_sha"]; !ok {
		t.Fatal("expected original_commit_sha preserved")
	}
}
