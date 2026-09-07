// Package lightsouts fetches motorsport session data from api.lightsouts.com.
package lightsouts

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

const (
	APIBase        = "https://api.lightsouts.com/v1"
	RequestTimeout = 30 * time.Second
	UserAgent      = "racing-mqtt/1.0 (https://github.com/resnostyle/racing-mqtt)"
)

// Client fetches series data from the Lightsouts API with ETag caching.
type Client struct {
	httpClient *http.Client
	etag       map[string]string
	cached     map[string]map[string]any
	mu         sync.Mutex
}

func NewClient() *Client {
	return &Client{
		httpClient: &http.Client{Timeout: RequestTimeout},
		etag:       make(map[string]string),
		cached:     make(map[string]map[string]any),
	}
}

// FetchSeries returns flattened sessions for a series slug.
func (c *Client) FetchSeries(ctx context.Context, slug string) ([]Session, error) {
	url := APIBase + "/series/" + slug
	payload, err := c.fetchJSON(ctx, url)
	if err != nil {
		return nil, fmt.Errorf("fetch series %s: %w", slug, err)
	}
	return FlattenSeries(payload), nil
}

// FetchSeriesSlugs fetches multiple series in parallel (bounded concurrency).
func (c *Client) FetchSeriesSlugs(ctx context.Context, slugs []string) ([]Session, error) {
	if len(slugs) == 0 {
		return nil, nil
	}
	type result struct {
		sessions []Session
		err      error
	}
	ch := make(chan result, len(slugs))
	sem := make(chan struct{}, 2)
	var wg sync.WaitGroup
	for _, slug := range slugs {
		wg.Add(1)
		go func(slug string) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()
			sessions, err := c.FetchSeries(ctx, slug)
			ch <- result{sessions: sessions, err: err}
		}(slug)
	}
	wg.Wait()
	close(ch)

	var out []Session
	var firstErr error
	for r := range ch {
		if r.err != nil {
			if firstErr == nil {
				firstErr = r.err
			}
			continue
		}
		out = append(out, r.sessions...)
	}
	if len(out) == 0 && firstErr != nil {
		return nil, firstErr
	}
	return out, nil
}

func (c *Client) fetchJSON(ctx context.Context, url string) (map[string]any, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "application/json")

	c.mu.Lock()
	if etag := c.etag[url]; etag != "" {
		req.Header.Set("If-None-Match", etag)
	}
	c.mu.Unlock()

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotModified {
		c.mu.Lock()
		payload := c.cached[url]
		c.mu.Unlock()
		if payload != nil {
			return payload, nil
		}
		return nil, fmt.Errorf("304 without cached payload for %s", url)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return nil, fmt.Errorf("HTTP %d: %s", resp.StatusCode, string(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	var payload map[string]any
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, err
	}

	c.mu.Lock()
	if etag := resp.Header.Get("ETag"); etag != "" {
		c.etag[url] = etag
	}
	c.cached[url] = payload
	c.mu.Unlock()

	return payload, nil
}
