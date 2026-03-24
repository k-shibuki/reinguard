package githubapi

import (
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

// GetJSON performs GET u with GitHub REST headers.
func (c *Client) GetJSON(ctx context.Context, u string, out any) error {
	if c.HTTP == nil {
		c.HTTP = http.DefaultClient
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

		resp, err := c.HTTP.Do(req)
		if err != nil {
			lastErr = err
			if attempt < maxAttempts-1 {
				time.Sleep(backoff)
				backoff *= 2
			}
			continue
		}
		body, _ := io.ReadAll(resp.Body)
		_ = resp.Body.Close()
		if resp.StatusCode == http.StatusTooManyRequests && attempt < maxAttempts-1 {
			lastErr = fmt.Errorf("429 from GitHub")
			time.Sleep(retryAfterDelay(resp, backoff))
			backoff *= 2
			continue
		}
		if isGitHubRateLimit403(resp, body) && attempt < maxAttempts-1 {
			lastErr = fmt.Errorf("403 rate limited from GitHub")
			time.Sleep(retryAfterDelay(resp, backoff))
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

func retryAfterDelay(resp *http.Response, fallback time.Duration) time.Duration {
	if resp == nil {
		return fallback
	}
	if ra := resp.Header.Get("Retry-After"); ra != "" {
		if sec, err := strconv.ParseInt(ra, 10, 64); err == nil && sec > 0 {
			return time.Duration(sec) * time.Second
		}
	}
	return fallback
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
