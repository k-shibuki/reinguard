package githubapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"
)

// Client is a minimal GitHub REST client with 429 backoff (ADR-0006: Bearer token from caller).
type Client struct {
	HTTP    *http.Client
	Token   string
	BaseURL string // optional override for tests (e.g. httptest server); default api.github.com
}

// APIBase returns the REST base URL without trailing slash.
func (c *Client) APIBase() string {
	if c.BaseURL != "" {
		s := strings.TrimRight(c.BaseURL, "/")
		return s
	}
	return "https://api.github.com"
}

// graphQLEndpoint returns the GraphQL HTTP URL for this client.
// For GitHub.com the REST root is https://api.github.com and GraphQL is /graphql on the same host.
// GitHub Enterprise Server often configures REST as https://HOST/api/v3; its GraphQL endpoint is
// https://HOST/api/graphql (not .../api/v3/graphql). Unsupported or custom layouts may need a
// different api_base; the primary supported target remains github.com (ADR-0006).
func (c *Client) graphQLEndpoint() string {
	base := c.APIBase()
	if strings.HasSuffix(base, "/api/v3") {
		return strings.TrimSuffix(base, "/api/v3") + "/api/graphql"
	}
	return base + "/graphql"
}

// GetJSON performs GET u with GitHub REST headers.
func (c *Client) GetJSON(ctx context.Context, u string, out any) error {
	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	const maxAttempts = 4
	var lastErr error
	backoff := 300 * time.Millisecond
	for attempt := 0; attempt < maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.Token)
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
			if attempt < maxAttempts-1 {
				if err := sleepCtx(ctx, backoff); err != nil {
					return err
				}
				backoff *= 2
			}
			continue
		}
		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("read response body: %w", readErr)
			if attempt < maxAttempts-1 {
				if err := sleepCtx(ctx, backoff); err != nil {
					return err
				}
				backoff *= 2
			}
			continue
		}
		if resp.StatusCode == http.StatusTooManyRequests && attempt < maxAttempts-1 {
			lastErr = fmt.Errorf("429 from GitHub")
			if err := sleepCtx(ctx, retryAfterDelay(resp, backoff)); err != nil {
				return err
			}
			backoff *= 2
			continue
		}
		if isGitHubRateLimit403(resp, body) && attempt < maxAttempts-1 {
			lastErr = fmt.Errorf("403 rate limited from GitHub")
			if err := sleepCtx(ctx, retryAfterDelay(resp, backoff)); err != nil {
				return err
			}
			backoff *= 2
			continue
		}
		if resp.StatusCode >= 400 {
			return fmt.Errorf("github %s: %s: %s", req.URL.String(), resp.Status, strings.TrimSpace(string(body)))
		}
		if out != nil {
			if err := json.Unmarshal(body, out); err != nil {
				return fmt.Errorf("decode json: %w", err)
			}
		}
		return nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("github request failed")
	}
	return lastErr
}

// graphqlEnvelope is the GitHub GraphQL HTTP JSON body (data + optional errors).
type graphqlEnvelope struct {
	Data   json.RawMessage        `json:"data"`
	Errors []graphqlErrorFragment `json:"errors"`
}

type graphqlErrorFragment struct {
	Message string `json:"message"`
}

// PostGraphQL posts a GraphQL request to the host's GraphQL URL (see graphQLEndpoint) with the
// same auth and 429 retry behavior as GetJSON (ADR-0006). If the response includes GraphQL-level
// errors, PostGraphQL returns a non-nil error after a successful HTTP status.
// The `data` field is unmarshaled into `out` when `out` is non-nil.
func (c *Client) PostGraphQL(ctx context.Context, query string, variables map[string]any, out any) error {
	if c == nil {
		return fmt.Errorf("nil client")
	}
	payload := map[string]any{"query": query}
	if len(variables) > 0 {
		payload["variables"] = variables
	}
	bodyBytes, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("graphql encode body: %w", err)
	}
	u := c.graphQLEndpoint()

	httpClient := c.HTTP
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	const maxAttempts = 4
	var lastErr error
	backoff := 300 * time.Millisecond
	for attempt := 0; attempt < maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(bodyBytes))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.Token)
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

		resp, err := httpClient.Do(req)
		if err != nil {
			lastErr = err
			if attempt < maxAttempts-1 {
				if err := sleepCtx(ctx, backoff); err != nil {
					return err
				}
				backoff *= 2
			}
			continue
		}
		body, readErr := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if readErr != nil {
			lastErr = fmt.Errorf("read response body: %w", readErr)
			if attempt < maxAttempts-1 {
				if err := sleepCtx(ctx, backoff); err != nil {
					return err
				}
				backoff *= 2
			}
			continue
		}
		if resp.StatusCode == http.StatusTooManyRequests && attempt < maxAttempts-1 {
			lastErr = fmt.Errorf("429 from GitHub")
			if err := sleepCtx(ctx, retryAfterDelay(resp, backoff)); err != nil {
				return err
			}
			backoff *= 2
			continue
		}
		if isGitHubRateLimit403(resp, body) && attempt < maxAttempts-1 {
			lastErr = fmt.Errorf("403 rate limited from GitHub")
			if err := sleepCtx(ctx, retryAfterDelay(resp, backoff)); err != nil {
				return err
			}
			backoff *= 2
			continue
		}
		if resp.StatusCode >= 400 {
			return fmt.Errorf("github graphql %s: %s: %s", req.URL.String(), resp.Status, strings.TrimSpace(string(body)))
		}
		var env graphqlEnvelope
		if err := json.Unmarshal(body, &env); err != nil {
			return fmt.Errorf("decode graphql json: %w", err)
		}
		if len(env.Errors) > 0 {
			return fmt.Errorf("graphql error: %s", env.Errors[0].Message)
		}
		if out != nil && len(env.Data) > 0 && string(env.Data) != "null" {
			if err := json.Unmarshal(env.Data, out); err != nil {
				return fmt.Errorf("decode graphql data: %w", err)
			}
		}
		return nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("github graphql request failed")
	}
	return lastErr
}

func retryAfterDelay(resp *http.Response, fallback time.Duration) time.Duration {
	if resp == nil {
		return fallback
	}
	if ra := resp.Header.Get("Retry-After"); ra != "" {
		if sec, err := strconv.ParseInt(ra, 10, 64); err == nil && sec > 0 {
			return time.Duration(sec) * time.Second
		}
	}
	// GitHub often omits Retry-After but sets X-RateLimit-Reset (epoch seconds) when remaining is 0.
	rem := resp.Header.Get("X-RateLimit-Remaining")
	if rem == "0" || (rem == "" && resp.StatusCode == http.StatusTooManyRequests) {
		if reset := resp.Header.Get("X-RateLimit-Reset"); reset != "" {
			if unixSec, err := strconv.ParseInt(reset, 10, 64); err == nil {
				if d := time.Until(time.Unix(unixSec, 0)); d > 0 {
					return d
				}
			}
		}
	}
	return fallback
}

func sleepCtx(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}

func isGitHubRateLimit403(resp *http.Response, body []byte) bool {
	if resp == nil || resp.StatusCode != 403 {
		return false
	}
	if resp.Header.Get("Retry-After") != "" {
		return true
	}
	b := strings.ToLower(string(body))
	return strings.Contains(b, "rate limit") || strings.Contains(b, "too many requests")
}
