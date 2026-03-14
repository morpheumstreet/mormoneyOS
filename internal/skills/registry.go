// Package skills implements ClawHub registry client for skill installation.
package skills

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const (
	defaultRegistryURL  = "https://clawhub.ai"
	defaultTimeoutSec   = 30
	wellKnownPath       = "/.well-known/clawhub.json"
	apiSkillPath        = "/api/v1/skills"
	apiDownloadPath     = "/api/v1/download"
	apiSearchPath       = "/api/v1/search"
)

// versionInfo holds version from registry.
type versionInfo struct {
	Version string `json:"version"`
}

// RegistryMeta holds skill metadata from ClawHub.
type RegistryMeta struct {
	Slug          string       `json:"slug"`
	DisplayName   string       `json:"displayName"`
	Summary       string       `json:"summary"`
	LatestVersion *versionInfo `json:"latestVersion"`
}

// SkillResponse is the API response for GET /api/v1/skills/{slug}.
type SkillResponse struct {
	Skill         RegistryMeta `json:"skill"`
	LatestVersion *versionInfo `json:"latestVersion"`
	Moderation *struct {
		IsMalwareBlocked bool `json:"isMalwareBlocked"`
		IsSuspicious     bool `json:"isSuspicious"`
	} `json:"moderation"`
}

// RegistryClient fetches skill metadata and zip bundles from ClawHub.
type RegistryClient struct {
	BaseURL    string
	HTTPClient *http.Client
}

// NewRegistryClient creates a client for the given registry URL.
func NewRegistryClient(registryURL string, timeoutSec int) *RegistryClient {
	base := strings.TrimSuffix(registryURL, "/")
	if base == "" {
		base = defaultRegistryURL
	}
	if timeoutSec <= 0 {
		timeoutSec = defaultTimeoutSec
	}
	return &RegistryClient{
		BaseURL: base,
		HTTPClient: &http.Client{
			Timeout: time.Duration(timeoutSec) * time.Second,
		},
	}
}

// Resolve fetches skill metadata from the registry.
func (c *RegistryClient) Resolve(ctx context.Context, slug string) (*RegistryMeta, string, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, "", fmt.Errorf("slug required")
	}
	if strings.Contains(slug, "..") || strings.Contains(slug, "/") || strings.Contains(slug, "\\") {
		return nil, "", fmt.Errorf("invalid slug: %s", slug)
	}
	u := c.BaseURL + apiSkillPath + "/" + url.PathEscape(slug)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, "", err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, "", fmt.Errorf("registry request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusNotFound {
		return nil, "", fmt.Errorf("skill not found: %s", slug)
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, "", fmt.Errorf("registry returned %d: %s", resp.StatusCode, string(body))
	}
	var sr SkillResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, "", fmt.Errorf("parse registry response: %w", err)
	}
	if sr.Moderation != nil && sr.Moderation.IsMalwareBlocked {
		return nil, "", fmt.Errorf("skill %s is blocked as malware", slug)
	}
	version := ""
	if sr.LatestVersion != nil {
		version = sr.LatestVersion.Version
	}
	if sr.Skill.LatestVersion != nil && version == "" {
		version = sr.Skill.LatestVersion.Version
	}
	if version == "" {
		return nil, "", fmt.Errorf("no version found for skill %s", slug)
	}
	meta := &RegistryMeta{
		Slug:        sr.Skill.Slug,
		DisplayName: sr.Skill.DisplayName,
		Summary:     sr.Skill.Summary,
		LatestVersion: sr.Skill.LatestVersion,
	}
	if meta.Slug == "" {
		meta.Slug = slug
	}
	if meta.DisplayName == "" {
		meta.DisplayName = slug
	}
	if meta.LatestVersion == nil && sr.LatestVersion != nil {
		meta.LatestVersion = sr.LatestVersion
	}
	return meta, version, nil
}

// doWithRetry runs fn and retries on 429 (rate limit) with exponential backoff.
func doWithRetry(ctx context.Context, maxAttempts int, fn func() (*http.Response, error)) (*http.Response, error) {
	var lastErr error
	for attempt := 0; attempt < maxAttempts; attempt++ {
		resp, err := fn()
		if err != nil {
			return nil, err
		}
		if resp.StatusCode != http.StatusTooManyRequests {
			return resp, nil
		}
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 256))
		resp.Body.Close()
		lastErr = fmt.Errorf("rate limit exceeded: %s", string(body))
		// Retry-After header (seconds) or exponential backoff: 3s, 6s, 12s
		delay := 3 * (1 << attempt)
		if ra := resp.Header.Get("Retry-After"); ra != "" {
			if sec, err := parseInt(ra); err == nil && sec > 0 && sec < 60 {
				delay = sec
			}
		}
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(time.Duration(delay) * time.Second):
			// retry
		}
	}
	return nil, lastErr
}

func parseInt(s string) (int, error) {
	var n int
	_, err := fmt.Sscanf(s, "%d", &n)
	return n, err
}

// Download fetches the skill zip bundle from the registry.
// Retries on 429 (rate limit) with exponential backoff.
func (c *RegistryClient) Download(ctx context.Context, slug, version string) ([]byte, error) {
	slug = strings.TrimSpace(slug)
	if slug == "" {
		return nil, fmt.Errorf("slug required")
	}
	if strings.Contains(slug, "..") || strings.Contains(slug, "/") || strings.Contains(slug, "\\") {
		return nil, fmt.Errorf("invalid slug: %s", slug)
	}
	u := c.BaseURL + apiDownloadPath + "?slug=" + url.QueryEscape(slug)
	if version != "" {
		u += "&version=" + url.QueryEscape(version)
	}
	resp, err := doWithRetry(ctx, 3, func() (*http.Response, error) {
		req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
		if err != nil {
			return nil, err
		}
		return c.HTTPClient.Do(req)
	})
	if err != nil {
		return nil, fmt.Errorf("download: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("download returned %d: %s", resp.StatusCode, string(body))
	}
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read download: %w", err)
	}
	return data, nil
}

// SearchResult is a skill from registry search.
type SearchResult struct {
	Slug        string  `json:"slug"`
	DisplayName string  `json:"displayName"`
	Summary     string  `json:"summary"`
	Version     *string `json:"version"`
	Score       float64 `json:"score"`
}

// SearchResponse is the API response for GET /api/v1/search.
type SearchResponse struct {
	Results []SearchResult `json:"results"`
}

// Search searches the registry by query.
func (c *RegistryClient) Search(ctx context.Context, query string, limit int) ([]SearchResult, error) {
	u := c.BaseURL + apiSearchPath + "?q=" + url.QueryEscape(query)
	if limit > 0 {
		u += "&limit=" + fmt.Sprint(limit)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("search request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("search returned %d: %s", resp.StatusCode, string(body))
	}
	var sr SearchResponse
	if err := json.NewDecoder(resp.Body).Decode(&sr); err != nil {
		return nil, fmt.Errorf("parse search response: %w", err)
	}
	return sr.Results, nil
}

// ListItem is a skill from registry list (GET /api/v1/skills).
type ListItem struct {
	Slug        string  `json:"slug"`
	DisplayName string  `json:"displayName"`
	Summary     string  `json:"summary"`
	Tags        map[string]string `json:"tags"`
	Stats       *struct {
		Downloads int `json:"downloads"`
		Stars     int `json:"stars"`
	} `json:"stats"`
	LatestVersion *versionInfo `json:"latestVersion"`
}

// ListResponse is the API response for GET /api/v1/skills.
type ListResponse struct {
	Items      []ListItem `json:"items"`
	NextCursor string     `json:"nextCursor"`
}

// List lists skills from the registry (paginated).
func (c *RegistryClient) List(ctx context.Context, cursor string, limit int) (*ListResponse, error) {
	u := c.BaseURL + apiSkillPath
	params := url.Values{}
	if cursor != "" {
		params.Set("cursor", cursor)
	}
	if limit > 0 {
		params.Set("limit", fmt.Sprint(limit))
	}
	if len(params) > 0 {
		u += "?" + params.Encode()
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, u, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Accept", "application/json")
	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("list request: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		return nil, fmt.Errorf("list returned %d: %s", resp.StatusCode, string(body))
	}
	var lr struct {
		Items      []ListItem `json:"items"`
		NextCursor string     `json:"nextCursor"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&lr); err != nil {
		return nil, fmt.Errorf("parse list response: %w", err)
	}
	return &ListResponse{Items: lr.Items, NextCursor: lr.NextCursor}, nil
}
