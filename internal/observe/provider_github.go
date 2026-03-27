package observe

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/k-shibuki/reinguard/internal/githubapi"
	"github.com/k-shibuki/reinguard/internal/observe/github/ci"
	"github.com/k-shibuki/reinguard/internal/observe/github/issues"
	"github.com/k-shibuki/reinguard/internal/observe/github/pullrequests"
	"github.com/k-shibuki/reinguard/internal/observe/github/reviews"
)

// GitHubProvider aggregates GitHub facets (ADR-0006).
type GitHubProvider struct {
	HTTPClient *http.Client
	// APIBase optionally overrides the GitHub REST root (tests / httptest).
	APIBase string
}

// NewGitHubProvider returns a GitHub aggregate provider.
func NewGitHubProvider() *GitHubProvider {
	return &GitHubProvider{HTTPClient: &http.Client{Timeout: 30 * time.Second}}
}

// GitHubProviderFactory builds a GitHub provider from config options (ADR-0009).
// Supported keys: api_base (string) — absolute http(s) REST API root override for tests or enterprise endpoints.
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
	return p, nil
}

// ID implements Provider.
func (*GitHubProvider) ID() string { return "github" }

// Collect implements Provider.
func (p *GitHubProvider) Collect(ctx context.Context, opts Options) (Fragment, error) {
	httpClient := p.HTTPClient
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 30 * time.Second}
	}
	token, err := githubapi.TokenFromGH(ctx, opts.WorkDir)
	if err != nil {
		return Fragment{
			Degraded: true,
			Diagnostics: []Diagnostic{{
				Severity: "error",
				Message:  err.Error(),
				Provider: "github",
				Code:     "auth_failed",
			}},
		}, nil
	}
	owner, repo, err := githubapi.RepoFromGH(ctx, opts.WorkDir)
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

	client := &githubapi.Client{HTTP: httpClient, Token: token, BaseURL: p.APIBase}
	signals := map[string]any{
		"repository": map[string]any{"owner": owner, "name": repo},
	}
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
	p.githubCollectIssues(ctx, client, owner, repo, wantFacet("issues"), signals, &diags, &degraded)
	p.githubCollectPullRequestsAndPRNum(ctx, client, owner, repo, opts.WorkDir, wantFacet, signals, appendWarnings, &diags, &degraded, &prNum)
	p.githubCollectCI(ctx, client, owner, repo, opts.WorkDir, wantFacet("ci"), signals, appendWarnings, &diags, &degraded)
	p.githubCollectReviews(ctx, client, owner, repo, prNum, wantFacet("reviews"), signals, &diags, &degraded)

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

func (*GitHubProvider) githubCollectPullRequestsAndPRNum(ctx context.Context, client *githubapi.Client, owner, repo, workDir string, wantFacet func(string) bool, signals map[string]any, appendWarnings func(string, []string), diags *[]Diagnostic, degraded *bool, prNum *int) {
	if !wantFacet("pull-requests") && !wantFacet("reviews") {
		return
	}
	m, warns, err := pullrequests.Collect(ctx, client, owner, repo, workDir)
	if err != nil {
		*degraded = true
		*diags = append(*diags, Diagnostic{Severity: "error", Message: err.Error(), Provider: "github.pull-requests"})
		return
	}
	appendWarnings("github.pull-requests", warns)
	if wantFacet("pull-requests") {
		mergeSignals(signals, m)
	}
	if prMap, ok := m["pull_requests"].(map[string]any); ok {
		*prNum = intFromMap(prMap, "pr_number_for_branch")
	}
}

func (*GitHubProvider) githubCollectCI(ctx context.Context, client *githubapi.Client, owner, repo, workDir string, want bool, signals map[string]any, appendWarnings func(string, []string), diags *[]Diagnostic, degraded *bool) {
	if !want {
		return
	}
	m, warns, err := ci.Collect(ctx, client, owner, repo, workDir)
	if err != nil {
		*degraded = true
		*diags = append(*diags, Diagnostic{Severity: "error", Message: err.Error(), Provider: "github.ci"})
		return
	}
	appendWarnings("github.ci", warns)
	mergeSignals(signals, m)
}

func (*GitHubProvider) githubCollectReviews(ctx context.Context, client *githubapi.Client, owner, repo string, prNum int, want bool, signals map[string]any, diags *[]Diagnostic, degraded *bool) {
	if !want {
		return
	}
	m, err := reviews.Collect(ctx, client, owner, repo, prNum)
	if err != nil {
		*degraded = true
		*diags = append(*diags, Diagnostic{Severity: "error", Message: err.Error(), Provider: "github.reviews"})
		return
	}
	mergeSignals(signals, m)
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
