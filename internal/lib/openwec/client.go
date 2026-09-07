package openwec

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"
)

const (
	DefaultBaseURL = "https://api.openwec.com/api/v1"
	RequestTimeout = 30 * time.Second
	UserAgent      = "racing-mqtt/1.0 (https://github.com/resnostyle/racing-mqtt)"
)

// Client calls the OpenWEC REST API.
type Client struct {
	baseURL    string
	apiKey     string
	httpClient *http.Client

	mu    sync.Mutex
	cache map[string]cacheEntry
}

type cacheEntry struct {
	body []byte
	at   time.Time
}

func NewClient(baseURL, apiKey string) *Client {
	baseURL = strings.TrimRight(strings.TrimSpace(baseURL), "/")
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	return &Client{
		baseURL:    baseURL,
		apiKey:     strings.TrimSpace(apiKey),
		httpClient: &http.Client{Timeout: RequestTimeout},
		cache:      make(map[string]cacheEntry),
	}
}

func (c *Client) ListEvents(ctx context.Context, seriesKey string, year int) ([]Event, error) {
	path := fmt.Sprintf("%s/series/%s/seasons/%d/events", c.baseURL, seriesKey, year)
	var out []Event
	if err := c.getJSON(ctx, path, &out, 6*time.Hour); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) ListSessions(ctx context.Context, seriesKey string, year, eventID int) ([]Session, error) {
	path := fmt.Sprintf("%s/series/%s/seasons/%d/events/%d/sessions", c.baseURL, seriesKey, year, eventID)
	var out []Session
	if err := c.getJSON(ctx, path, &out, 6*time.Hour); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) GetResults(ctx context.Context, sessionID int) ([]Result, error) {
	path := fmt.Sprintf("%s/sessions/%d/results", c.baseURL, sessionID)
	var out []Result
	if err := c.getJSON(ctx, path, &out, 0); err != nil {
		return nil, err
	}
	return out, nil
}

func (c *Client) getJSON(ctx context.Context, url string, dest any, cacheTTL time.Duration) error {
	c.mu.Lock()
	if cacheTTL > 0 {
		if entry, ok := c.cache[url]; ok && time.Since(entry.at) < cacheTTL {
			body := entry.body
			c.mu.Unlock()
			return json.Unmarshal(body, dest)
		}
	}
	c.mu.Unlock()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("User-Agent", UserAgent)
	req.Header.Set("Accept", "application/json")
	if c.apiKey != "" {
		req.Header.Set("X-API-Key", c.apiKey)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("openwec HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if cacheTTL > 0 {
		c.mu.Lock()
		c.cache[url] = cacheEntry{body: append([]byte(nil), body...), at: time.Now()}
		c.mu.Unlock()
	}
	return json.Unmarshal(body, dest)
}

// SeriesKeyForLightsouts maps a Lightsouts series slug to an OpenWEC series key.
func SeriesKeyForLightsouts(slug string) (string, bool) {
	switch strings.ToLower(strings.TrimSpace(slug)) {
	case "wec":
		return "WEC", true
	case "imsa-sportscar-championship", "imsa":
		return "IMSA", true
	default:
		return "", false
	}
}
