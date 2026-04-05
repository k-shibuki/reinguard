package observe

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/k-shibuki/reinguard/internal/githubapi"
	"github.com/k-shibuki/reinguard/internal/observe/github/ci"
	"github.com/k-shibuki/reinguard/internal/observe/github/issues"
	"github.com/k-shibuki/reinguard/internal/observe/github/prquery"
	"github.com/k-shibuki/reinguard/internal/observe/github/pullrequests"
)

var botReviewerIDPattern = regexp.MustCompile(`^[a-z0-9_]+$`)

// GitHubProvider aggregates GitHub facets (ADR-0006).
type GitHubProvider struct {
	HTTPClient *http.Client
	// APIBase optionally overrides the GitHub REST root (tests / httptest).
	APIBase string
	// BotReviewers configures optional PR comment / review status observation per bot (P2-1).
	BotReviewers []prquery.BotReviewer
}

// NewGitHubProvider returns a GitHub aggregate provider.
func NewGitHubProvider() *GitHubProvider {
	return &GitHubProvider{HTTPClient: &http.Client{Timeout: 30 * time.Second}}
}

// GitHubProviderFactory builds a GitHub provider from config options (ADR-0009).
// Supported keys: api_base (string); bot_reviewers (array of { id, login, required, enrich? }).
func GitHubProviderFactory(opts map[string]any) (Provider, error) {
	p := NewGitHubProvider()
	if len(opts) == 0 {
		return p, nil
	}
	if raw, exists := opts["api_base"]; exists {
		v, ok := raw.(string)
		if !ok {
			return nil, fmt.Errorf("github provider: options.api_base must be a string")
		}
		s := strings.TrimSpace(v)
		if s == "" {
			return nil, fmt.Errorf("github provider: options.api_base must be non-empty when set")
		}
		u, err := url.Parse(s)
		if err != nil || u.Scheme == "" || u.Host == "" {
			return nil, fmt.Errorf("github provider: options.api_base must be an absolute URL with scheme and host")
		}
		if u.Scheme != "http" && u.Scheme != "https" {
			return nil, fmt.Errorf("github provider: options.api_base scheme must be http or https")
		}
		p.APIBase = s
	}
	tr, err := parseBotReviewers(opts)
	if err != nil {
		return nil, err
	}
	p.BotReviewers = tr
	return p, nil
}

func parseBotReviewers(opts map[string]any) ([]prquery.BotReviewer, error) {
	raw, ok := opts["bot_reviewers"]
	if !ok {
		return nil, nil
	}
	arr, ok := raw.([]any)
	if !ok {
		return nil, fmt.Errorf("github provider: options.bot_reviewers must be an array")
	}
	out := make([]prquery.BotReviewer, 0, len(arr))
	seenID := make(map[string]struct{}, len(arr))
	for i, item := range arr {
		m, ok := item.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("github provider: options.bot_reviewers[%d] must be an object", i)
		}
		id, _ := m["id"].(string)
		id = strings.TrimSpace(id)
		if id == "" {
			return nil, fmt.Errorf("github provider: options.bot_reviewers[%d].id is required", i)
		}
		if !botReviewerIDPattern.MatchString(id) {
			return nil, fmt.Errorf("github provider: options.bot_reviewers[%d].id must match ^[a-z0-9_]+$", i)
		}
		if _, dup := seenID[id]; dup {
			return nil, fmt.Errorf("github provider: options.bot_reviewers: duplicate id %q", id)
		}
		seenID[id] = struct{}{}

		login, _ := m["login"].(string)
		login = strings.TrimSpace(login)
		if login == "" {
			return nil, fmt.Errorf("github provider: options.bot_reviewers[%d].login is required", i)
		}
		required, err := parseBotReviewerRequiredFlag(i, m)
		if err != nil {
			return nil, err
		}
		enrich, err := parseBotReviewerEnrichList(i, m)
		if err != nil {
			return nil, err
		}
		if err := prquery.ValidateEnrichmentNames(enrich); err != nil {
			return nil, fmt.Errorf("github provider: options.bot_reviewers[%d]: %w", i, err)
		}
		out = append(out, prquery.BotReviewer{ID: id, Login: login, Enrich: enrich, Required: required})
	}
	return out, nil
}

func parseBotReviewerRequiredFlag(i int, m map[string]any) (bool, error) {
	reqRaw, ok := m["required"]
	if !ok {
		return false, fmt.Errorf("github provider: options.bot_reviewers[%d].required is required", i)
	}
	required, ok := reqRaw.(bool)
	if !ok {
		return false, fmt.Errorf("github provider: options.bot_reviewers[%d].required must be a boolean", i)
	}
	return required, nil
}

func parseBotReviewerEnrichList(i int, m map[string]any) ([]string, error) {
	switch e := m["enrich"].(type) {
	case nil:
		return nil, nil
	case []any:
		var enrich []string
		for _, x := range e {
			s, ok := x.(string)
			if !ok {
				return nil, fmt.Errorf("github provider: options.bot_reviewers[%d].enrich must contain only strings", i)
			}
			s = strings.TrimSpace(s)
			if s == "" {
				return nil, fmt.Errorf("github provider: options.bot_reviewers[%d].enrich contains empty string", i)
			}
			enrich = append(enrich, s)
		}
		return enrich, nil
	default:
		return nil, fmt.Errorf("github provider: options.bot_reviewers[%d].enrich must be an array of strings", i)
	}
}

// ID implements Provider.
func (*GitHubProvider) ID() string { return "github" }

// Collect implements Provider.
func (p *GitHubProvider) Collect(ctx context.Context, opts Options) (Fragment, error) {
	httpClient := p.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	identity, err := githubapi.ResolveGitHubRepoIdentityFromWorkDir(ctx, opts.WorkDir)
	if err != nil {
		return Fragment{
			Degraded: true,
			Diagnostics: []Diagnostic{{
				Severity: "error",
				Message:  err.Error(),
				Provider: "github",
				Code:     "repo_resolve_failed",
			}},
		}, nil
	}
	owner, repo := identity.Owner, identity.Name

	signals := map[string]any{
		"repository": map[string]any{
			"owner":           owner,
			"name":            repo,
			"identity_source": string(identity.Source),
		},
	}

	token, err := githubapi.TokenFromGH(ctx, opts.WorkDir)
	if err != nil {
		return Fragment{
			Signals:  signals,
			Degraded: true,
			Diagnostics: []Diagnostic{{
				Severity: "error",
				Message:  err.Error(),
				Provider: "github",
				Code:     "auth_failed",
			}},
		}, nil
	}

	client := &githubapi.Client{HTTP: httpClient, Token: token, BaseURL: p.APIBase}
	var diags []Diagnostic
	degraded := false

	appendWarnings := func(provider string, w []string) {
		for _, msg := range w {
			diags = append(diags, Diagnostic{Severity: "warning", Message: msg, Provider: provider})
			degraded = true
		}
	}

	wantFacet := func(name string) bool {
		if opts.GitHubFacet == "" {
			return true
		}
		return opts.GitHubFacet == name
	}

	var prNum int
	prLookupOK := true
	var prHeadSHA string
	explicitScope := opts.Scope.PRNumber > 0 || strings.TrimSpace(opts.Scope.Branch) != ""
	needScopedCIHead := wantFacet("ci") && explicitScope
	p.githubCollectIssues(ctx, client, owner, repo, wantFacet("issues"), signals, &diags, &degraded)
	p.githubCollectPullRequestsAndPRNum(ctx, client, owner, repo, opts, wantFacet, signals, appendWarnings, &diags, &degraded, &prNum, &prLookupOK)
	prHeadSHA = p.githubCollectPRGraph(ctx, client, owner, repo, prNum, wantFacet("pull-requests"), wantFacet("reviews"), needScopedCIHead, prLookupOK, signals, &diags, &degraded, opts)
	if wantFacet("ci") && explicitScope && strings.TrimSpace(prHeadSHA) == "" {
		degraded = true
		diags = append(diags, Diagnostic{
			Severity: "error",
			Message:  "scoped CI collection could not resolve the selected head SHA",
			Provider: "github.ci",
			Code:     "scoped_head_unresolved",
		})
		mergeSignals(signals, map[string]any{
			"ci": map[string]any{
				"ci_status": "unknown",
				"head_sha":  "",
			},
		})
	} else {
		p.githubCollectCI(ctx, client, owner, repo, opts.WorkDir, prHeadSHA, wantFacet("ci"), signals, appendWarnings, &diags, &degraded)
	}

	return Fragment{Signals: signals, Diagnostics: diags, Degraded: degraded}, nil
}

func (*GitHubProvider) githubCollectIssues(ctx context.Context, client *githubapi.Client, owner, repo string, want bool, signals map[string]any, diags *[]Diagnostic, degraded *bool) {
	if !want {
		return
	}
	m, err := issues.Collect(ctx, client, owner, repo)
	if err != nil {
		*degraded = true
		*diags = append(*diags, Diagnostic{Severity: "error", Message: err.Error(), Provider: "github.issues"})
		return
	}
	mergeSignals(signals, m)
}

func (*GitHubProvider) githubCollectPullRequestsAndPRNum(ctx context.Context, client *githubapi.Client, owner, repo string, opts Options, wantFacet func(string) bool, signals map[string]any, appendWarnings func(string, []string), diags *[]Diagnostic, degraded *bool, prNum *int, prLookupOK *bool) {
	explicitScope := opts.Scope.PRNumber > 0 || strings.TrimSpace(opts.Scope.Branch) != ""
	needPRForCI := wantFacet("ci") && explicitScope
	if !wantFacet("pull-requests") && !wantFacet("reviews") && !needPRForCI {
		return
	}
	m, _, warns, err := pullrequests.Collect(ctx, client, owner, repo, opts.WorkDir, pullrequests.ScopeOptions{
		Branch:   opts.Scope.Branch,
		PRNumber: opts.Scope.PRNumber,
	})
	if err != nil {
		*prLookupOK = false
		*degraded = true
		*diags = append(*diags, Diagnostic{Severity: "error", Message: err.Error(), Provider: "github.pull-requests"})
		return
	}
	*prLookupOK = true
	appendWarnings("github.pull-requests", warns)
	if wantFacet("pull-requests") {
		mergeSignals(signals, m)
	}
	if prMap, ok := m["pull_requests"].(map[string]any); ok {
		*prNum = intFromMap(prMap, "pr_number_for_branch")
	}
}

func (*GitHubProvider) githubCollectCI(ctx context.Context, client *githubapi.Client, owner, repo, workDir, headSHA string, want bool, signals map[string]any, appendWarnings func(string, []string), diags *[]Diagnostic, degraded *bool) {
	if !want {
		return
	}
	m, warns, err := ci.Collect(ctx, client, owner, repo, workDir, headSHA)
	if err != nil {
		*degraded = true
		*diags = append(*diags, Diagnostic{Severity: "error", Message: err.Error(), Provider: "github.ci"})
		return
	}
	appendWarnings("github.ci", warns)
	mergeSignals(signals, m)
}

func (p *GitHubProvider) githubCollectPRGraph(ctx context.Context, client *githubapi.Client, owner, repo string, prNum int, wantPull, wantRev, wantCI bool, prLookupOK bool, signals map[string]any, diags *[]Diagnostic, degraded *bool, opts Options) string {
	if !wantPull && !wantRev && !wantCI {
		return ""
	}
	if !prLookupOK || prNum <= 0 {
		return ""
	}
	pullDetail, revInner, err := prquery.Collect(ctx, client, owner, repo, prNum, p.BotReviewers)
	if err != nil {
		*degraded = true
		*diags = append(*diags, Diagnostic{Severity: "error", Message: err.Error(), Provider: "github.pr-query"})
		return ""
	}
	headSHA, _ := pullDetail["head_sha"].(string)
	headRef, _ := pullDetail["head_ref"].(string)
	if wantPull && len(pullDetail) > 0 {
		if existing, ok := signals["pull_requests"].(map[string]any); ok {
			for k, v := range pullDetail {
				existing[k] = v
			}
			updateObservedScope(existing, opts, prNum, headRef, headSHA)
		} else {
			pm := make(map[string]any, len(pullDetail))
			for k, v := range pullDetail {
				pm[k] = v
			}
			updateObservedScope(pm, opts, prNum, headRef, headSHA)
			signals["pull_requests"] = pm
		}
	}
	if wantRev {
		mergeSignals(signals, map[string]any{"reviews": revInner})
	}
	if wantCI {
		return headSHA
	}
	return ""
}

func mergeSignals(dst map[string]any, src map[string]any) {
	for k, v := range src {
		dst[k] = v
	}
}

func intFromMap(m map[string]any, key string) int {
	v, ok := m[key]
	if !ok {
		return 0
	}
	switch x := v.(type) {
	case int:
		return x
	case int64:
		return int(x)
	case float64:
		return int(x)
	default:
		return 0
	}
}

func updateObservedScope(prMap map[string]any, opts Options, prNum int, headRef, headSHA string) {
	raw, ok := prMap["observed_scope"].(map[string]any)
	if !ok {
		raw = map[string]any{}
		prMap["observed_scope"] = raw
	}
	raw["resolved_pr_number"] = prNum
	if headRef != "" {
		raw["pr_head_branch"] = headRef
		if opts.Scope.PRNumber > 0 {
			prMap["current_branch"] = headRef
			raw["effective_branch"] = headRef
		}
	}
	if headSHA != "" {
		raw["pr_head_sha"] = headSHA
	}
}
