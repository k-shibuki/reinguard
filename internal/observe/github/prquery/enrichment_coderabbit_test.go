package prquery

import "testing"

func TestCoderabbitEnrichment_tryAgainMinutesSeconds(t *testing.T) {
	t.Parallel()
	e := coderabbitEnrichment{}
	got := e.Enrich("Rate limit exceeded. Please try again in 5 minutes and 30 seconds")
	assertSeconds(t, got, 330)
}

func TestCoderabbitEnrichment_pleaseWaitMinutesSeconds(t *testing.T) {
	t.Parallel()
	e := coderabbitEnrichment{}
	body := "> [!WARNING]\n> ## Rate limit exceeded\n> Please wait **19 minutes and 47 seconds** before requesting another review.\n"
	got := e.Enrich(body)
	assertSeconds(t, got, 19*60+47)
}

func TestCoderabbitEnrichment_oneMinute(t *testing.T) {
	t.Parallel()
	e := coderabbitEnrichment{}
	got := e.Enrich("Please try again in 1 minute")
	assertSeconds(t, got, 60)
}

func TestCoderabbitEnrichment_secondsOnly(t *testing.T) {
	t.Parallel()
	e := coderabbitEnrichment{}
	got := e.Enrich("try again in 45 seconds")
	assertSeconds(t, got, 45)
}

func TestCoderabbitEnrichment_caseInsensitive(t *testing.T) {
	t.Parallel()
	e := coderabbitEnrichment{}
	got := e.Enrich("TRY AGAIN IN 5 MINUTES")
	assertSeconds(t, got, 300)
}

func TestCoderabbitEnrichment_zeroSecondsNoSignal(t *testing.T) {
	t.Parallel()
	e := coderabbitEnrichment{}
	if got := e.Enrich("try again in 0 seconds"); got != nil {
		t.Fatalf("got %v", got)
	}
}

func TestCoderabbitEnrichment_noMatch(t *testing.T) {
	t.Parallel()
	e := coderabbitEnrichment{}
	if got := e.Enrich("All good"); got != nil {
		t.Fatalf("got %v", got)
	}
}

func assertSeconds(t *testing.T, got map[string]any, want int) {
	t.Helper()
	if got == nil {
		t.Fatal("got nil")
	}
	sec, ok := got["rate_limit_remaining_seconds"].(int)
	if !ok || sec != want {
		t.Fatalf("got %v", got)
	}
}

func TestCoderabbitEnrichment_reviewStatusProcessing(t *testing.T) {
	t.Parallel()
	e := coderabbitEnrichment{}
	got := e.Enrich("## Review status\n**Status:** in progress\n")
	if !got["cr_review_processing"].(bool) {
		t.Fatalf("got %v", got)
	}
}

func TestCoderabbitEnrichment_reviewStatusCompleted(t *testing.T) {
	t.Parallel()
	e := coderabbitEnrichment{}
	got := e.Enrich("**Status:** ✅ completed\n")
	if v, ok := got["cr_review_processing"]; ok {
		processing, ok := v.(bool)
		if !ok || processing {
			t.Fatalf("want processing false/absent, got %v", got)
		}
	}
}

func TestCoderabbitEnrichment_reviewStatusCompletedClean(t *testing.T) {
	t.Parallel()
	e := coderabbitEnrichment{}
	got := e.Enrich("No issues found.\n")
	if got == nil {
		t.Fatal("got nil")
	}
	clean, ok := got["cr_review_completed_clean"].(bool)
	if !ok || !clean {
		t.Fatalf("want cr_review_completed_clean=true, got %v", got)
	}
}

func TestCoderabbitEnrichment_walkthroughAndCleanMarker(t *testing.T) {
	t.Parallel()
	e := coderabbitEnrichment{}
	got := e.Enrich("### Walkthrough\nNo additional issues found.\n")
	if got == nil {
		t.Fatal("got nil")
	}
	walkthrough, ok := got["cr_walkthrough_present"].(bool)
	if !ok || !walkthrough {
		t.Fatalf("want cr_walkthrough_present=true, got %v", got)
	}
	clean, ok := got["cr_review_completed_clean"].(bool)
	if !ok || !clean {
		t.Fatalf("want cr_review_completed_clean=true, got %v", got)
	}
}

func TestCoderabbitEnrichment_walkthroughMarker(t *testing.T) {
	t.Parallel()
	e := coderabbitEnrichment{}
	got := e.Enrich("### Walkthrough\nSome text")
	if !got["cr_walkthrough_present"].(bool) {
		t.Fatalf("got %v", got)
	}
}

func TestParseCoderabbitDuplicateCount(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		want int
	}{
		{name: "empty", body: "", want: 0},
		{name: "no_match", body: "Actionable comments posted: 4", want: 0},
		{name: "emoji_and_count", body: "**Actionable comments posted: 4**\n\n<details>\n<summary>♻️ Duplicate comments (2)</summary>", want: 2},
		{name: "recycling_symbol_no_vs", body: "\u267b Duplicate comments (1)", want: 1},
		{name: "malformed_non_numeric", body: "♻️ Duplicate comments (x)", want: 0},
		{name: "zero_count", body: "♻️ Duplicate comments (0)", want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := parseCoderabbitDuplicateCount(tt.body); got != tt.want {
				t.Fatalf("parseCoderabbitDuplicateCount(%q) = %d, want %d", tt.body, got, tt.want)
			}
		})
	}
}

func TestCoderabbitEnrichment_EnrichReviewBody(t *testing.T) {
	t.Parallel()
	e := coderabbitEnrichment{}
	body := "<summary>♻️ Duplicate comments (3)</summary>"
	got := e.EnrichReviewBody(body)
	if got == nil {
		t.Fatal("got nil")
	}
	if n, ok := got["cr_duplicate_findings_count"].(int); !ok || n != 3 {
		t.Fatalf("got %v", got)
	}
	if n, ok := got["duplicate_findings_count"].(int); !ok || n != 3 {
		t.Fatalf("got %v", got)
	}
	if e.EnrichReviewBody("no duplicate section") != nil {
		t.Fatal("want nil")
	}
}

func TestCoderabbitEnrichment_Enrich_providerNeutralIssueCommentAliases(t *testing.T) {
	t.Parallel()
	e := coderabbitEnrichment{}
	// Body triggers reviewed head SHA + in-progress processing; assert normalized keys only.
	body := "Reviewing files between [1c0a07a](https://example.com/1) and [4b680dbdeadbeef](https://example.com/2).\n" +
		"Currently processing new changes in this PR.\n"
	got := e.Enrich(body)
	if got == nil {
		t.Fatal("got nil")
	}
	if v, ok := got["review_processing"].(bool); !ok || !v {
		t.Fatalf("want review_processing=true, got %+v", got)
	}
	if sha, ok := got["reviewed_head_sha"].(string); !ok || sha != "4b680dbdeadbeef" {
		t.Fatalf("want reviewed_head_sha, got %+v", got)
	}
}

func TestCoderabbitEnrichment_ClassifyStatus_providerNeutralAliases(t *testing.T) {
	t.Parallel()
	e := coderabbitEnrichment{}
	if g := e.ClassifyStatus(map[string]any{"review_processing": true}); g != BotStatusPending {
		t.Fatalf("ClassifyStatus(review_processing)= %q want %q", g, BotStatusPending)
	}
	if g := e.ClassifyStatus(map[string]any{"review_completed_clean": true}); g != BotStatusCompletedClean {
		t.Fatalf("ClassifyStatus(review_completed_clean)= %q want %q", g, BotStatusCompletedClean)
	}
	if g := e.ClassifyStatus(map[string]any{"walkthrough_present": true, "latest_comment_at": ""}); g != BotStatusPending {
		t.Fatalf("ClassifyStatus(walkthrough_present)= %q want %q", g, BotStatusPending)
	}
}

func TestCoderabbitEnrichment_EnrichReviewBody_actionableAndOutside(t *testing.T) {
	t.Parallel()
	e := coderabbitEnrichment{}
	//nolint:govet // test table readability over struct packing
	tests := []struct {
		name string
		body string
		want map[string]int
	}{
		{
			name: "actionable_and_outside",
			body: "Actionable review comments (2)\nComments outside the diff range (1)\n",
			want: map[string]int{
				"cr_actionable_comments_count": 2, "actionable_findings_count": 2,
				"cr_outside_diff_comments_count": 1, "outside_diff_findings_count": 1,
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			got := e.EnrichReviewBody(tt.body)
			if got == nil {
				t.Fatal("got nil")
			}
			for k, w := range tt.want {
				n, ok := got[k].(int)
				if !ok || n != w {
					t.Fatalf("%s: got[%q]=%v want %d full=%+v", tt.name, k, got[k], w, got)
				}
			}
		})
	}
}

func TestParseCoderabbitActionableCommentsCount(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		want int
	}{
		{name: "empty", body: "", want: 0},
		{name: "match", body: "Actionable review comments (4)", want: 4},
		{name: "older_header", body: "**Actionable comments posted: 4**", want: 4},
		{name: "no_match", body: "no actionable line", want: 0},
		{name: "zero", body: "Actionable review comments (0)", want: 0},
		{name: "malformed", body: "Actionable review comments (x)", want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := parseCoderabbitActionableCommentsCount(tt.body); got != tt.want {
				t.Fatalf("parseCoderabbitActionableCommentsCount(%q) = %d, want %d", tt.body, got, tt.want)
			}
		})
	}
}

func TestParseCoderabbitOutsideDiffCommentsCount(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name string
		body string
		want int
	}{
		{name: "canonical", body: "Comments outside the diff range (3)", want: 3},
		{name: "short_form", body: "outside the diff (2)", want: 2},
		{name: "no_match", body: "outside section missing", want: 0},
		{name: "malformed", body: "Comments outside the diff range (x)", want: 0},
		{name: "zero", body: "Comments outside the diff range (0)", want: 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			if got := parseCoderabbitOutsideDiffCommentsCount(tt.body); got != tt.want {
				t.Fatalf("parseCoderabbitOutsideDiffCommentsCount(%q) = %d, want %d", tt.body, got, tt.want)
			}
		})
	}
}

func TestIsCoderabbitFindingConversationComment(t *testing.T) {
	t.Parallel()
	if !IsCoderabbitFindingConversationComment("Please fix the nil dereference in handler.go.") {
		t.Fatal("generic tier-0 body should count as finding-shaped")
	}
	if IsCoderabbitFindingConversationComment("No issues found.\n") {
		t.Fatal("terminal clean marker is excluded")
	}
	if IsCoderabbitFindingConversationComment("### Walkthrough\n") {
		t.Fatal("walkthrough is excluded")
	}
	if IsCoderabbitFindingConversationComment("Sure! I'll kick off a new review to verify the fixes.") {
		t.Fatal("plain operational acknowledgement is excluded")
	}
	wrappedFinding := "<!-- This is an auto-generated comment: summarize by coderabbit.ai -->\n" +
		"<details>\n<summary>⚠️ Outside diff range comments (1)</summary><blockquote>\n" +
		"`internal/observe/github/prquery/prquery.go`: _⚠️ Potential issue_\n</blockquote></details>"
	if !IsCoderabbitFindingConversationComment(wrappedFinding) {
		t.Fatal("wrapped outside-diff summary should still count as finding")
	}
	// Operational / status PR comments share tier 6 with CoderabbitIssueCommentMaxTier; they must not
	// inflate finding_conversation_comments_count (merge-readiness non-thread signal).
	ops := []string{
		"Rate limit exceeded. Please try again in 5 minutes and 30 seconds",
		"Currently processing new changes in this PR. This may take a few minutes, please wait...",
		"Review paused until you comment.",
		"Review failed: head commit changed.",
		"<!-- This is an auto-generated comment: summarize by coderabbit.ai -->\n<!-- This is an auto-generated comment: review paused by coderabbit.ai -->\n\n> [!NOTE]\n> ## Reviews paused",
	}
	for _, body := range ops {
		if IsCoderabbitFindingConversationComment(body) {
			t.Fatalf("operational body should be excluded: %q", body)
		}
		if CoderabbitIssueCommentMaxTier(body) != 6 {
			t.Fatalf("sanity: want tier 6 for operational sample: %q", body)
		}
	}
}

func TestCoderabbitIssueCommentMaxTier_decisiveStatusesShareTierSix(t *testing.T) {
	t.Parallel()
	cases := []string{
		"Rate limit exceeded. Please try again in 1 minute",
		"Review paused until you comment.",
		"Review failed: head commit changed.",
		"No issues found.",
		"**Status:** ✅ completed\n",
		"**Status:** in progress\n",
	}
	for _, body := range cases {
		if got := CoderabbitIssueCommentMaxTier(body); got != 6 {
			t.Fatalf("CoderabbitIssueCommentMaxTier(%q) = %d, want 6", body, got)
		}
	}
	if got := CoderabbitIssueCommentMaxTier("<!-- This is an auto-generated reply by CodeRabbit -->\nReview triggered.\n"); got != 0 {
		t.Fatalf("wrapper-only operational ack want tier 0, got %d", got)
	}
	if got := CoderabbitIssueCommentMaxTier("<!-- This is an auto-generated comment: summarize by coderabbit.ai -->\n### Walkthrough\n"); got != 1 {
		t.Fatalf("wrapped walkthrough-only want tier 1, got %d", got)
	}
	if got := CoderabbitIssueCommentMaxTier("### Walkthrough\n"); got != 1 {
		t.Fatalf("walkthrough-only want tier 1, got %d", got)
	}
	if got := CoderabbitIssueCommentMaxTier("Some generic comment text."); got != 0 {
		t.Fatalf("no-markers want tier 0, got %d", got)
	}
}

func TestCoderabbitEnrichment_ClassifyStatus_order(t *testing.T) {
	t.Parallel()
	e := coderabbitEnrichment{}
	// Processing is evaluated before active cooldown; both signals need not conflict.
	if g := e.ClassifyStatus(map[string]any{"contains_rate_limit": true, "cr_review_processing": true}); g != BotStatusPending {
		t.Fatalf("got %q", g)
	}
	if g := e.ClassifyStatus(map[string]any{"rate_limit_remaining_seconds": 300, "cr_review_processing": true}); g != BotStatusPending {
		t.Fatalf("processing wins over active cooldown: got %q", g)
	}
	if g := e.ClassifyStatus(map[string]any{"rate_limit_remaining_seconds": 300}); g != BotStatusRateLimited {
		t.Fatalf("got %q", g)
	}
	if g := e.ClassifyStatus(map[string]any{"cr_review_processing": true}); g != BotStatusPending {
		t.Fatalf("got %q", g)
	}
	if g := e.ClassifyStatus(map[string]any{"cr_review_completed_clean": true}); g != BotStatusCompletedClean {
		t.Fatalf("got %q", g)
	}
	if g := e.ClassifyStatus(map[string]any{"has_review": true, "cr_review_completed_clean": true}); g != BotStatusCompletedClean {
		t.Fatalf("got %q", g)
	}
	if g := e.ClassifyStatus(map[string]any{"has_review": true}); g != BotStatusCompleted {
		t.Fatalf("got %q", g)
	}
	if g := e.ClassifyStatus(map[string]any{"cr_walkthrough_present": true, "latest_comment_at": "2026-01-01T00:00:00Z"}); g != BotStatusPending {
		t.Fatalf("got %q", g)
	}
}

func TestCoderabbitEnrichment_ClassifyStatus_staleRateLimitTextDoesNotOverrideClean(t *testing.T) {
	t.Parallel()
	e := coderabbitEnrichment{}
	// Same issue-comment body can mention "rate limit" historically while reporting a terminal clean summary.
	m := map[string]any{
		"contains_rate_limit":          true,
		"review_completed_clean":       true,
		"rate_limit_remaining_seconds": 0,
		"latest_comment_at":            "2026-04-06T00:31:05Z",
	}
	if g := e.ClassifyStatus(m); g != BotStatusCompletedClean {
		t.Fatalf("got %q want completed_clean", g)
	}
	s, basis := classifyCoderabbitStatusWithBasis(m)
	if s != BotStatusCompletedClean || basis != "review_completed_clean" {
		t.Fatalf("basis got %q %q", s, basis)
	}
}

func TestCoderabbitEnrichment_noActionableCommentsClean(t *testing.T) {
	t.Parallel()
	e := coderabbitEnrichment{}
	got := e.Enrich("No actionable comments were generated in the recent review.\n")
	if got == nil {
		t.Fatal("got nil")
	}
	clean, ok := got["cr_review_completed_clean"].(bool)
	if !ok || !clean {
		t.Fatalf("want cr_review_completed_clean=true, got %v", got)
	}
}

func TestCoderabbitEnrichment_reviewedHeadSHAFromRange(t *testing.T) {
	t.Parallel()
	tests := []struct {
		name    string
		body    string
		wantSHA string
	}{
		{
			name:    "range_extracts_trailing_sha",
			body:    "Reviewing files that changed from the base of the PR and between [1c0a07a](https://example.com/1) and [4b680dbdeadbeef](https://example.com/2).\n",
			wantSHA: "4b680dbdeadbeef",
		},
		{
			name:    "single_sha_at_bracket_form",
			body:    "Reviewing files that changed from the base of the PR at [4b680dbdeadbeef](https://example.com/2).\n",
			wantSHA: "4b680dbdeadbeef",
		},
		{
			name:    "walkthrough_only_no_reviewed_head_sha",
			body:    "### Walkthrough\nGeneral notes without bracket SHAs.\n",
			wantSHA: "",
		},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			e := coderabbitEnrichment{}
			got := e.Enrich(tt.body)
			if tt.wantSHA == "" {
				if got != nil && got["cr_reviewed_head_sha"] != nil {
					t.Fatalf("want no reviewed sha, got %v", got)
				}
				return
			}
			if got == nil {
				t.Fatal("got nil")
			}
			sha, ok := got["cr_reviewed_head_sha"].(string)
			if !ok || sha != tt.wantSHA {
				t.Fatalf("want cr_reviewed_head_sha=%s, got %v", tt.wantSHA, got)
			}
		})
	}
}
