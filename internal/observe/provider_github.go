package observe

import (
	"context"
	"net/http"
	"time"

	"github.com/k-shibuki/reinguard/internal/githubapi"
	"github.com/k-shibuki/reinguard/internal/observe/github/ci"
	"github.com/k-shibuki/reinguard/internal/observe/github/issues"
	"github.com/k-shibuki/reinguard/internal/observe/github/pullrequests"
)

// GitHubProvider aggregates GitHub REST facets (ADR-0006).
type GitHubProvider struct {
	HTTPClient *http.Client
	// APIBase optionally overrides the GitHub REST root (tests / httptest).
	APIBase string
}

// NewGitHubProvider returns a GitHub aggregate provider.
func NewGitHubProvider() *GitHubProvider {
	return &GitHubProvider{HTTPClient: &http.Client{Timeout: 30 * time.Second}}
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

	if wantFacet("issues") {
		if m, err := issues.Collect(ctx, client, owner, repo); err != nil {
			degraded = true
			diags = append(diags, Diagnostic{Severity: "error", Message: err.Error(), Provider: "github.issues"})
		} else {
			mergeGitHubSignals(signals, m)
		}
	}

	if wantFacet("pull-requests") {
		if m, warns, err := pullrequests.Collect(ctx, client, owner, repo, opts.WorkDir); err != nil {
			degraded = true
			diags = append(diags, Diagnostic{Severity: "error", Message: err.Error(), Provider: "github.pull-requests"})
		} else {
			appendWarnings("github.pull-requests", warns)
			mergeGitHubSignals(signals, m)
		}
	}

	if wantFacet("ci") {
		if m, warns, err := ci.Collect(ctx, client, owner, repo, opts.WorkDir); err != nil {
			degraded = true
			diags = append(diags, Diagnostic{Severity: "error", Message: err.Error(), Provider: "github.ci"})
		} else {
			appendWarnings("github.ci", warns)
			mergeGitHubSignals(signals, m)
		}
	}

	return Fragment{Signals: signals, Diagnostics: diags, Degraded: degraded}, nil
}

func mergeGitHubSignals(dst map[string]any, src map[string]any) {
	for k, v := range src {
		dst[k] = v
	}
}
