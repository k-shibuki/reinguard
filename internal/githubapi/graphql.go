package githubapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type gqlRequestBody struct {
	Variables map[string]any `json:"variables,omitempty"`
	Query     string         `json:"query"`
}

// PostGraphQL calls the GitHub GraphQL API (same host as APIBase).
func (c *Client) PostGraphQL(ctx context.Context, query string, variables map[string]any, out any) error {
	if c.HTTP == nil {
		c.HTTP = http.DefaultClient
	}
	payload, err := json.Marshal(gqlRequestBody{Query: query, Variables: variables})
	if err != nil {
		return err
	}
	u := c.APIBase() + "/graphql"
	const maxAttempts = 4
	var lastErr error
	backoff := 300 * time.Millisecond
	for attempt := 0; attempt < maxAttempts; attempt++ {
		req, err := http.NewRequestWithContext(ctx, http.MethodPost, u, bytes.NewReader(payload))
		if err != nil {
			return err
		}
		req.Header.Set("Authorization", "Bearer "+c.Token)
		req.Header.Set("Accept", "application/vnd.github+json")
		req.Header.Set("Content-Type", "application/json")
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
			lastErr = fmt.Errorf("429 from GitHub GraphQL")
			time.Sleep(backoff)
			backoff *= 2
			continue
		}
		if resp.StatusCode >= 400 {
			return fmt.Errorf("github graphql: %s: %s", resp.Status, strings.TrimSpace(string(body)))
		}
		if out != nil {
			if err := json.Unmarshal(body, out); err != nil {
				return fmt.Errorf("decode graphql json: %w", err)
			}
		}
		return nil
	}
	if lastErr == nil {
		lastErr = fmt.Errorf("github graphql request failed")
	}
	return lastErr
}
