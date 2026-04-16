package pullrequests

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os/exec"
	"strings"
	"testing"

	"github.com/k-shibuki/reinguard/internal/githubapi"
)

func TestCollect_prForBranch_usesPullsListExactHead(t *testing.T) {
	t.Parallel()
	// Given: repo on branch feature and API listing open PR with head o:feature
	dir := t.TempDir()
	run(t, "git", dir, "init")
	run(t, "git", dir, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init")
	run(t, "git", dir, "branch", "-M", "feature")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search/issues":
			_, _ = w.Write([]byte(`{"total_count":0,"incomplete_results":false,"items":[]}`))
		case "/repos/o/r/pulls":
			if r.URL.Query().Get("state") != "open" || r.URL.Query().Get("head") != "o:feature" {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write([]byte(`[{"number":99,"head":{"ref":"feature"}}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	c := &githubapi.Client{HTTP: srv.Client(), Token: "tok", BaseURL: srv.URL}
	// When: Collect runs
	m, scope, warns, err := Collect(context.Background(), c, "o", "r", dir, ScopeOptions{}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 0 {
		t.Fatalf("%v", warns)
	}
	pr, ok := m["pull_requests"].(map[string]any)
	if !ok {
		t.Fatalf("expected pull_requests map, got %T", m["pull_requests"])
	}
	// Then: pr_exists_for_branch and pr number 99
	if !pr["pr_exists_for_branch"].(bool) {
		t.Fatalf("want pr for branch, got %v", pr)
	}
	if pr["pr_number_for_branch"].(int) != 99 {
		t.Fatalf("want pr number 99, got %v", pr["pr_number_for_branch"])
	}
	if scope.Selection != SelectionCurrentBranch {
		t.Fatalf("scope=%+v", scope)
	}
}

func TestCollect_withGitRepo(t *testing.T) {
	t.Parallel()
	// Given: main branch, empty PR list
	dir := t.TempDir()
	run(t, "git", dir, "init")
	run(t, "git", dir, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init")
	run(t, "git", dir, "branch", "-M", "main")

	emptySearch := `{"total_count":0,"incomplete_results":false,"items":[]}`
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search/issues":
			_, _ = w.Write([]byte(emptySearch))
		case "/repos/o/r/pulls":
			if r.URL.Query().Get("state") != "open" || r.URL.Query().Get("head") != "o:main" {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write([]byte(`[]`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	c := &githubapi.Client{HTTP: srv.Client(), Token: "tok", BaseURL: srv.URL}
	// When: Collect runs
	m, scope, warns, err := Collect(context.Background(), c, "o", "r", dir, ScopeOptions{}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 0 {
		t.Fatalf("%v", warns)
	}
	pr, ok := m["pull_requests"].(map[string]any)
	if !ok {
		t.Fatalf("expected pull_requests map, got %T", m["pull_requests"])
	}
	// Then: current_branch main, open_count 0
	if pr["current_branch"].(string) != "main" {
		t.Fatalf("%v", pr)
	}
	if oc, ok := pr["open_count"].(int); !ok || oc != 0 {
		t.Fatalf("open_count want 0 got %v (%T)", pr["open_count"], pr["open_count"])
	}
	if scope.LocalBranch != "main" {
		t.Fatalf("scope=%+v", scope)
	}
}

func run(t *testing.T, name, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("%s %v: %v %s", name, args, err, string(out))
	}
}

func TestCollect_detachedHeadWarning(t *testing.T) {
	t.Parallel()
	// Given: detached HEAD checkout
	dir := t.TempDir()
	run(t, "git", dir, "init")
	run(t, "git", dir, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init")
	commit, err := exec.Command("git", "-C", dir, "rev-parse", "HEAD").Output()
	if err != nil {
		t.Fatal(err)
	}
	run(t, "git", dir, "checkout", strings.TrimSpace(string(commit)))

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"total_count":0,"incomplete_results":false,"items":[]}`))
	}))
	t.Cleanup(srv.Close)
	c := &githubapi.Client{HTTP: srv.Client(), Token: "tok", BaseURL: srv.URL}
	// When: Collect runs
	_, _, warns, err := Collect(context.Background(), c, "o", "r", dir, ScopeOptions{}, "")
	if err != nil {
		t.Fatal(err)
	}
	// Then: non-empty warnings (detached head)
	if len(warns) == 0 {
		t.Fatal("expected warning")
	}
}

func TestCollect_explicitBranchOverridesLocalBranch(t *testing.T) {
	t.Parallel()
	// Given: local repo on main and an API response for feature/review
	dir := t.TempDir()
	run(t, "git", dir, "init")
	run(t, "git", dir, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init")
	run(t, "git", dir, "branch", "-M", "main")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search/issues":
			_, _ = w.Write([]byte(`{"total_count":1,"items":[{"number":17}]}`))
		case "/repos/o/r/pulls":
			if r.URL.Query().Get("head") != "o:feature/review" {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write([]byte(`[{"number":17,"head":{"ref":"feature/review"}}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	c := &githubapi.Client{HTTP: srv.Client(), Token: "tok", BaseURL: srv.URL}
	// When: Collect runs with an explicit branch override
	m, scope, warns, err := Collect(context.Background(), c, "o", "r", dir, ScopeOptions{Branch: "feature/review"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 0 {
		t.Fatalf("%v", warns)
	}
	// Then: the resolved PR and observed scope follow the requested branch
	pr, ok := m["pull_requests"].(map[string]any)
	if !ok {
		t.Fatalf("expected pull_requests map, got %T", m["pull_requests"])
	}
	if pr["current_branch"] != "feature/review" || pr["pr_number_for_branch"].(int) != 17 {
		t.Fatalf("pull_requests=%+v", pr)
	}
	observed, ok := pr["observed_scope"].(map[string]any)
	if !ok {
		t.Fatalf("expected observed_scope map, got %T", pr["observed_scope"])
	}
	if observed["local_branch_at_collect"] != "main" || observed["requested_branch"] != "feature/review" {
		t.Fatalf("observed_scope=%+v", observed)
	}
	if scope.Selection != SelectionExplicitBranch {
		t.Fatalf("scope=%+v", scope)
	}
}

func TestCollect_explicitPROpensRequestedPR(t *testing.T) {
	t.Parallel()
	// Given: local repo on main and an explicit open PR target
	dir := t.TempDir()
	run(t, "git", dir, "init")
	run(t, "git", dir, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init")
	run(t, "git", dir, "branch", "-M", "main")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search/issues":
			_, _ = w.Write([]byte(`{"total_count":2,"items":[{"number":17},{"number":18}]}`))
		case "/repos/o/r/pulls/18":
			_, _ = w.Write([]byte(`{"number":18,"state":"open","head":{"ref":"feature/scoped"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	c := &githubapi.Client{HTTP: srv.Client(), Token: "tok", BaseURL: srv.URL}
	// When: Collect runs with an explicit PR number
	m, scope, warns, err := Collect(context.Background(), c, "o", "r", dir, ScopeOptions{PRNumber: 18}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 0 {
		t.Fatalf("%v", warns)
	}
	// Then: the observed scope resolves to PR #18 and its head branch
	pr, ok := m["pull_requests"].(map[string]any)
	if !ok {
		t.Fatalf("expected pull_requests map, got %T", m["pull_requests"])
	}
	if !pr["pr_exists_for_branch"].(bool) || pr["pr_number_for_branch"].(int) != 18 {
		t.Fatalf("pull_requests=%+v", pr)
	}
	observed, ok := pr["observed_scope"].(map[string]any)
	if !ok {
		t.Fatalf("expected observed_scope map, got %T", pr["observed_scope"])
	}
	reqPR, ok := observed["requested_pr_number"].(int)
	if !ok || reqPR != 18 || observed["effective_branch"] != "feature/scoped" {
		t.Fatalf("observed_scope=%+v", observed)
	}
	if scope.Selection != SelectionExplicitPR || scope.EffectiveBranch != "feature/scoped" {
		t.Fatalf("scope=%+v", scope)
	}
}

func TestCollect_summaryViewAddsRESTPullMetadata(t *testing.T) {
	t.Parallel()
	dir := t.TempDir()
	run(t, "git", dir, "init")
	run(t, "git", dir, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init")
	run(t, "git", dir, "branch", "-M", "main")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search/issues":
			_, _ = w.Write([]byte(`{"total_count":2,"items":[{"number":17},{"number":18}]}`))
		case "/repos/o/r/pulls/18":
			_, _ = w.Write([]byte(`{
				"number":18,
				"state":"open",
				"title":"feat: summary pull metadata",
				"draft":false,
				"mergeable":true,
				"mergeable_state":"clean",
				"base":{"ref":"main"},
				"head":{
					"ref":"feature/scoped",
					"sha":"0123456789abcdef0123456789abcdef01234567",
					"repo":{"name":"fork-repo","owner":{"login":"fork-owner"}}
				},
				"labels":[{"name":"feat"}]
			}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	c := &githubapi.Client{HTTP: srv.Client(), Token: "tok", BaseURL: srv.URL}
	m, _, warns, err := Collect(context.Background(), c, "o", "r", dir, ScopeOptions{PRNumber: 18}, ViewSummary)
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 0 {
		t.Fatalf("%v", warns)
	}
	pr, ok := m["pull_requests"].(map[string]any)
	if !ok {
		t.Fatalf("expected pull_requests map, got %T", m["pull_requests"])
	}
	if pr["head_sha"] != "0123456789abcdef0123456789abcdef01234567" {
		t.Fatalf("pull_requests=%+v", pr)
	}
	if pr["mergeable"] != "mergeable" || pr["merge_state_status"] != "clean" {
		t.Fatalf("pull_requests=%+v", pr)
	}
	if pr["head_repo_owner"] != "fork-owner" || pr["head_repo_name"] != "fork-repo" {
		t.Fatalf("pull_requests=%+v", pr)
	}
	labels, ok := pr["labels"].([]any)
	if !ok || len(labels) != 1 || labels[0] != "feat" {
		t.Fatalf("labels=%+v", pr["labels"])
	}
}

func TestCollect_explicitPRNotFoundReturnsError(t *testing.T) {
	t.Parallel()
	// Given: local repo on main and an explicit PR target that the API cannot find
	dir := t.TempDir()
	run(t, "git", dir, "init")
	run(t, "git", dir, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init")
	run(t, "git", dir, "branch", "-M", "main")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search/issues":
			_, _ = w.Write([]byte(`{"total_count":0,"items":[]}`))
		case "/repos/o/r/pulls/99":
			http.NotFound(w, r)
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	c := &githubapi.Client{HTTP: srv.Client(), Token: "tok", BaseURL: srv.URL}
	// When: Collect runs with an unknown explicit PR number
	_, _, _, err := Collect(context.Background(), c, "o", "r", dir, ScopeOptions{PRNumber: 99}, "")

	// Then: the missing PR surfaces as an error
	if err == nil || !strings.Contains(err.Error(), "404") {
		t.Fatalf("got %v", err)
	}
}

func TestCollect_explicitPRClosedReturnsError(t *testing.T) {
	t.Parallel()
	// Given: local repo on main and an explicit PR target that is closed
	dir := t.TempDir()
	run(t, "git", dir, "init")
	run(t, "git", dir, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init")
	run(t, "git", dir, "branch", "-M", "main")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search/issues":
			_, _ = w.Write([]byte(`{"total_count":1,"items":[{"number":18}]}`))
		case "/repos/o/r/pulls/18":
			_, _ = w.Write([]byte(`{"number":18,"state":"closed","head":{"ref":"feature/scoped"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	c := &githubapi.Client{HTTP: srv.Client(), Token: "tok", BaseURL: srv.URL}
	// When: Collect runs with a closed explicit PR number
	_, _, _, err := Collect(context.Background(), c, "o", "r", dir, ScopeOptions{PRNumber: 18}, "")

	// Then: the helper rejects the non-open PR
	if err == nil || !strings.Contains(err.Error(), "not open") {
		t.Fatalf("got %v", err)
	}
}

func TestCollect_explicitBranchWithoutMatchingPROmitsResolution(t *testing.T) {
	t.Parallel()
	// Given: local repo on main and an explicit branch with no matching open PR
	dir := t.TempDir()
	run(t, "git", dir, "init")
	run(t, "git", dir, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init")
	run(t, "git", dir, "branch", "-M", "main")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search/issues":
			_, _ = w.Write([]byte(`{"total_count":1,"items":[{"number":17}]}`))
		case "/repos/o/r/pulls":
			if r.URL.Query().Get("head") != "o:feature/missing" {
				http.NotFound(w, r)
				return
			}
			_, _ = w.Write([]byte(`[]`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	c := &githubapi.Client{HTTP: srv.Client(), Token: "tok", BaseURL: srv.URL}
	// When: Collect runs with an explicit branch that has no matching PR
	m, scope, warns, err := Collect(context.Background(), c, "o", "r", dir, ScopeOptions{Branch: "feature/missing"}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 0 {
		t.Fatalf("%v", warns)
	}

	// Then: the branch remains observed without resolving a PR number
	pr, ok := m["pull_requests"].(map[string]any)
	if !ok {
		t.Fatalf("expected pull_requests map, got %T", m["pull_requests"])
	}
	if pr["current_branch"] != "feature/missing" || pr["pr_exists_for_branch"].(bool) {
		t.Fatalf("pull_requests=%+v", pr)
	}
	if pr["pr_number_for_branch"].(int) != 0 {
		t.Fatalf("pull_requests=%+v", pr)
	}
	if scope.Selection != SelectionExplicitBranch || scope.ResolvedPRNumber != 0 {
		t.Fatalf("scope=%+v", scope)
	}
}

func TestCollect_explicitPRWinsOverRequestedBranchForEffectiveBranch(t *testing.T) {
	t.Parallel()
	// Given: local repo on main, explicit PR number, and a different explicit branch override
	dir := t.TempDir()
	run(t, "git", dir, "init")
	run(t, "git", dir, "-c", "user.email=t@t", "-c", "user.name=t", "commit", "--allow-empty", "-m", "init")
	run(t, "git", dir, "branch", "-M", "main")

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/search/issues":
			_, _ = w.Write([]byte(`{"total_count":1,"items":[{"number":18}]}`))
		case "/repos/o/r/pulls/18":
			_, _ = w.Write([]byte(`{"number":18,"state":"open","head":{"ref":"feature/scoped"}}`))
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)

	c := &githubapi.Client{HTTP: srv.Client(), Token: "tok", BaseURL: srv.URL}
	// When: Collect runs with both explicit selectors
	m, scope, warns, err := Collect(context.Background(), c, "o", "r", dir, ScopeOptions{
		Branch:   "feature/ignored",
		PRNumber: 18,
	}, "")
	if err != nil {
		t.Fatal(err)
	}
	if len(warns) != 0 {
		t.Fatalf("%v", warns)
	}

	// Then: explicit PR selection wins and current_branch follows the PR head ref
	pr, ok := m["pull_requests"].(map[string]any)
	if !ok {
		t.Fatalf("expected pull_requests map, got %T", m["pull_requests"])
	}
	if pr["current_branch"] != "feature/scoped" || pr["pr_number_for_branch"].(int) != 18 {
		t.Fatalf("pull_requests=%+v", pr)
	}
	observed, ok := pr["observed_scope"].(map[string]any)
	if !ok {
		t.Fatalf("expected observed_scope map, got %T", pr["observed_scope"])
	}
	if observed["requested_branch"] != "feature/ignored" || observed["effective_branch"] != "feature/scoped" {
		t.Fatalf("observed_scope=%+v", observed)
	}
	if scope.Selection != SelectionExplicitPR || scope.EffectiveBranch != "feature/scoped" {
		t.Fatalf("scope=%+v", scope)
	}
}
