package drift

import (
	"context"
	"encoding/xml"
	"fmt"
	"io"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	formulaDriftYouTubeChannelID  = "UCLqU1MEK55LYkV3M8W8DbhQ"
	formulaDriftYouTubeChannelURL = "https://www.youtube.com/formuladrift"
	formulaDriftYouTubeLiveURL    = "https://www.youtube.com/formuladrift/live"
	formulaDriftYouTubeFeedURL    = "https://www.youtube.com/feeds/videos.xml?channel_id=" + formulaDriftYouTubeChannelID
)

var hashtagSearchTerms = map[string][]string{
	"FDLV":   {"las vegas", "high stakes"},
	"FDLB":   {"long beach"},
	"FDSEA":  {"seattle"},
	"FDINDY": {"indianapolis", "indy"},
	"FDCT":   {"stafford", "connecticut"},
	"FDATL":  {"atlanta"},
	"FDORL":  {"orlando"},
}

type youtubeVideo struct {
	ID        string
	Title     string
	Published time.Time
	URL       string
}

type youtubeFeed struct {
	Entries []youtubeFeedEntry `xml:"entry"`
}

type youtubeFeedEntry struct {
	Title     string `xml:"title"`
	VideoID   string `xml:"http://www.youtube.com/xml/schemas/2015 videoId"`
	Published string `xml:"published"`
}

// enrichYouTubeURLs adds Formula DRIFT YouTube links to schedule events.
func enrichYouTubeURLs(ctx context.Context, httpClient *http.Client, events []Event, now time.Time) {
	if len(events) == 0 {
		return
	}
	videos, err := fetchYouTubeFeed(ctx, httpClient)
	if err != nil {
		videos = nil
	}
	for i := range events {
		e := &events[i]
		if e.YouTubeChannelURL == "" {
			e.YouTubeChannelURL = formulaDriftYouTubeChannelURL
		}
		if e.YouTubeLiveURL == "" {
			e.YouTubeLiveURL = formulaDriftYouTubeLiveURL
		}
		best := matchYouTubeVideo(*e, videos, now)
		e.YouTubeURL = pickYouTubeURL(*e, e.YouTubeURL, best, now)
	}
}

func fetchYouTubeFeed(ctx context.Context, httpClient *http.Client) ([]youtubeVideo, error) {
	if httpClient == nil {
		httpClient = &http.Client{Timeout: requestTimeout}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, formulaDriftYouTubeFeedURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", formuladUserAgent)
	req.Header.Set("Accept", "application/atom+xml, application/xml, text/xml")
	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("youtube feed HTTP %d", resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	return parseYouTubeFeed(body)
}

func parseYouTubeFeed(body []byte) ([]youtubeVideo, error) {
	var feed youtubeFeed
	if err := xml.Unmarshal(body, &feed); err != nil {
		return nil, err
	}
	out := make([]youtubeVideo, 0, len(feed.Entries))
	for _, entry := range feed.Entries {
		if entry.VideoID == "" {
			continue
		}
		published, err := time.Parse(time.RFC3339, entry.Published)
		if err != nil {
			continue
		}
		out = append(out, youtubeVideo{
			ID:        entry.VideoID,
			Title:     entry.Title,
			Published: published.UTC(),
			URL:       "https://www.youtube.com/watch?v=" + entry.VideoID,
		})
	}
	return out, nil
}

func eventSearchTerms(e Event) []string {
	seen := map[string]struct{}{}
	add := func(parts ...string) {
		for _, part := range parts {
			part = strings.TrimSpace(strings.ToLower(part))
			if part == "" {
				continue
			}
			if _, ok := seen[part]; ok {
				continue
			}
			seen[part] = struct{}{}
		}
	}
	add(strings.ReplaceAll(e.Slug, "-", " "))
	add(strings.ToLower(e.Name))
	if e.Venue != "" {
		add(e.Venue)
		parts := strings.Fields(strings.ToLower(e.Venue))
		if len(parts) >= 2 {
			add(strings.Join(parts[:2], " "))
		}
	}
	if terms, ok := hashtagSearchTerms[e.Hashtag]; ok {
		add(terms...)
	}
	terms := make([]string, 0, len(seen))
	for term := range seen {
		terms = append(terms, term)
	}
	sort.Strings(terms)
	return terms
}

func scoreYouTubeVideo(e Event, v youtubeVideo, now time.Time) int {
	title := strings.ToLower(v.Title)
	if strings.Contains(title, "fdj") {
		return -1
	}
	year := strconv.Itoa(e.Start.Year())
	if !strings.Contains(title, year) && !strings.Contains(title, "formula drift") {
		return -1
	}

	score := 0
	matched := false
	for _, term := range eventSearchTerms(e) {
		if term != "" && strings.Contains(title, term) {
			score += 10
			matched = true
		}
	}
	if !matched {
		return -1
	}

	windowStart := e.Start.Add(-21 * 24 * time.Hour)
	windowEnd := e.End.Add(7 * 24 * time.Hour)
	if v.Published.Before(windowStart) || v.Published.After(windowEnd) {
		score -= 5
	} else {
		score += 3
	}

	if strings.Contains(title, "teaser") {
		score += 4
	}
	for _, kw := range []string{"top 32", "top 16", "qualifying", "final four", "final"} {
		if strings.Contains(title, kw) {
			score += 6
		}
	}
	for _, kw := range []string{"highlight", "inside clips", "podium podcast", "drivers meeting"} {
		if strings.Contains(title, kw) {
			score -= 4
		}
	}

	if ActiveEvent([]Event{e}, now) != nil {
		for _, kw := range []string{"top 32", "top 16", "qualifying", "final"} {
			if strings.Contains(title, kw) {
				score += 8
			}
		}
	} else if now.Before(e.Start) && strings.Contains(title, "teaser") {
		score += 5
	}

	return score
}

func matchYouTubeVideo(e Event, videos []youtubeVideo, now time.Time) string {
	bestScore := 0
	var bestURL string
	for _, v := range videos {
		score := scoreYouTubeVideo(e, v, now)
		if score > bestScore {
			bestScore = score
			bestURL = v.URL
		}
	}
	if bestScore < 8 {
		return ""
	}
	return bestURL
}

func pickYouTubeURL(e Event, pageURL, matchedURL string, now time.Time) string {
	if pageURL != "" {
		return pageURL
	}
	if ActiveEvent([]Event{e}, now) != nil {
		return formulaDriftYouTubeLiveURL
	}
	if matchedURL != "" {
		return matchedURL
	}
	if now.Before(e.End) && now.After(e.Start.Add(-72*time.Hour)) {
		return formulaDriftYouTubeLiveURL
	}
	return formulaDriftYouTubeChannelURL
}

func parseYouTubeWatchURL(html string) string {
	for _, id := range youtubeWatchRE.FindAllStringSubmatch(html, -1) {
		if len(id) == 2 && id[1] != "" {
			return "https://www.youtube.com/watch?v=" + id[1]
		}
	}
	return ""
}
