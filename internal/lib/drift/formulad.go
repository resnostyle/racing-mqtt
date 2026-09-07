package drift

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/resnostyle/racing-mqtt/internal/lib/ics"
)

const (
	formuladScheduleURL = "https://www.formulad.com/schedule"
	formuladUserAgent   = "racing-mqtt/1.0 (https://github.com/resnostyle/racing-mqtt)"
	requestTimeout      = 30 * time.Second
)

var (
	scheduleLinkRE = regexp.MustCompile(`href="(/schedule/(\d{4})/([a-z0-9-]+))"`)
	roundRE        = regexp.MustCompile(`RD\s*(\d+)`)
	dateRangeRE    = regexp.MustCompile(`>([A-Z]{3})\s+(\d{1,2})\s*-\s*([A-Z]{3})\s+(\d{1,2})<`)
	locationRE     = regexp.MustCompile(`>([A-Za-z][A-Za-z\s,]+,\s*[A-Za-z\s]+,\s*USA)<`)
	hashtagRE      = regexp.MustCompile(`#(FD[A-Z0-9]+)`)
	youtubeWatchRE = regexp.MustCompile(`youtube\.com/watch\?v=([A-Za-z0-9_-]{11})`)
	titleRE        = regexp.MustCompile(`<title>([^|<]+)`)
)

// venueBySlug is the track name when the page does not expose one clearly.
var venueBySlug = map[string]string{
	"atlanta":      "Road Atlanta",
	"orlando":      "Orlando Speed World",
	"stafford":     "Stafford Motor Speedway",
	"indianapolis": "Indianapolis Motor Speedway",
	"seattle":      "Evergreen Speedway",
	"las-vegas":    "Las Vegas Motor Speedway",
	"long-beach":   "Streets of Long Beach",
	"long-beach2":  "Long Beach Shoreline",
}

var monthMap = map[string]time.Month{
	"JAN": time.January, "FEB": time.February, "MAR": time.March, "APR": time.April,
	"MAY": time.May, "JUN": time.June, "JUL": time.July, "AUG": time.August,
	"SEP": time.September, "OCT": time.October, "NOV": time.November, "DEC": time.December,
}

// FormuladFetcher loads the Formula DRIFT schedule from formulad.com.
type FormuladFetcher struct {
	httpClient  *http.Client
	scheduleURL string
}

func NewFormuladFetcher(scheduleURL string) *FormuladFetcher {
	if strings.TrimSpace(scheduleURL) == "" {
		scheduleURL = formuladScheduleURL
	}
	return &FormuladFetcher{
		httpClient:  &http.Client{Timeout: requestTimeout},
		scheduleURL: scheduleURL,
	}
}

func (f *FormuladFetcher) FetchEvents(ctx context.Context) ([]Event, error) {
	body, err := f.get(ctx, f.scheduleURL)
	if err != nil {
		return nil, err
	}
	seen := map[string]struct{}{}
	type link struct {
		year int
		slug string
	}
	var links []link
	for _, m := range scheduleLinkRE.FindAllStringSubmatch(string(body), -1) {
		year, _ := strconv.Atoi(m[2])
		slug := m[3]
		key := fmt.Sprintf("%d/%s", year, slug)
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		links = append(links, link{year: year, slug: slug})
	}
	if len(links) == 0 {
		return nil, fmt.Errorf("formulad: no schedule links found")
	}

	var events []Event
	for _, l := range links {
		ev, err := f.fetchEvent(ctx, l.year, l.slug)
		if err != nil {
			continue
		}
		events = append(events, ev)
	}
	if len(events) == 0 {
		return nil, fmt.Errorf("formulad: no events parsed")
	}
	enrichYouTubeURLs(ctx, f.httpClient, events, time.Now().UTC())
	sort.Slice(events, func(i, j int) bool {
		return events[i].Start.Before(events[j].Start)
	})
	return events, nil
}

func (f *FormuladFetcher) fetchEvent(ctx context.Context, year int, slug string) (Event, error) {
	url := fmt.Sprintf("https://www.formulad.com/schedule/%d/%s", year, slug)
	body, err := f.get(ctx, url)
	if err != nil {
		return Event{}, err
	}
	html := string(body)
	name := strings.TrimSpace(titleRE.FindStringSubmatch(html)[1])
	if idx := strings.Index(name, "|"); idx > 0 {
		name = strings.TrimSpace(name[:idx])
	}
	start, end, ok := parseDateRange(html, year)
	if !ok {
		return Event{}, fmt.Errorf("formulad: no dates for %s", slug)
	}
	round := 0
	if m := roundRE.FindStringSubmatch(html); len(m) == 2 {
		round, _ = strconv.Atoi(m[1])
	}
	location := ""
	if m := locationRE.FindStringSubmatch(html); len(m) == 2 {
		location = strings.TrimSpace(m[1])
	}
	venue := venueBySlug[slug]
	hashtag := ""
	if m := hashtagRE.FindStringSubmatch(html); len(m) == 2 {
		hashtag = m[1]
	}
	pageURL := parseYouTubeWatchURL(html)
	return Event{
		UID:               fmt.Sprintf("formulad:%d:%s", year, slug),
		Series:            "Formula Drift",
		Name:              name,
		Slug:              slug,
		Round:             round,
		Location:          location,
		Venue:             venue,
		Hashtag:           hashtag,
		Link:              url,
		YouTubeURL:        pageURL,
		YouTubeLiveURL:    formulaDriftYouTubeLiveURL,
		YouTubeChannelURL: formulaDriftYouTubeChannelURL,
		Start:             start,
		End:               end,
		Source:            "formulad",
	}, nil
}

func parseDateRange(html string, year int) (time.Time, time.Time, bool) {
	m := dateRangeRE.FindStringSubmatch(html)
	if len(m) != 5 {
		return time.Time{}, time.Time{}, false
	}
	startMonth, ok1 := monthMap[m[1]]
	endMonth, ok2 := monthMap[m[3]]
	if !ok1 || !ok2 {
		return time.Time{}, time.Time{}, false
	}
	startDay, _ := strconv.Atoi(m[2])
	endDay, _ := strconv.Atoi(m[4])
	start := time.Date(year, startMonth, startDay, 14, 0, 0, 0, time.UTC)
	// End is exclusive for weekend events: day after last day at same time.
	end := time.Date(year, endMonth, endDay, 23, 59, 0, 0, time.UTC).Add(time.Minute)
	return start, end, true
}

func (f *FormuladFetcher) get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", formuladUserAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml")
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("formulad HTTP %d for %s", resp.StatusCode, url)
	}
	return io.ReadAll(resp.Body)
}

// ICSFetcher loads events from an iCalendar URL and filters by series name.
type ICSFetcher struct {
	httpClient *http.Client
	url        string
	filter     []string
}

func NewICSFetcher(url string, filter ...string) *ICSFetcher {
	if len(filter) == 0 {
		filter = []string{"formula drift", "formulad"}
	}
	return &ICSFetcher{
		httpClient: &http.Client{Timeout: requestTimeout},
		url:        url,
		filter:     filter,
	}
}

func (f *ICSFetcher) FetchEvents(ctx context.Context) ([]Event, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, f.url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", formuladUserAgent)
	req.Header.Set("Accept", "text/calendar")
	resp, err := f.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if !strings.Contains(strings.ToLower(string(body[:min(64, len(body))])), "begin:vcalendar") {
		return nil, fmt.Errorf("ics: response is not a calendar")
	}
	parsed, err := ics.Parse(string(body))
	if err != nil {
		return nil, err
	}
	filtered := ics.FilterSummary(parsed, f.filter...)
	out := make([]Event, 0, len(filtered))
	for _, e := range filtered {
		end := e.End
		if e.AllDay && !end.IsZero() {
			end = end.Add(24 * time.Hour)
		}
		out = append(out, Event{
			UID:               e.UID,
			Series:            "Formula Drift",
			Name:              e.Summary,
			Slug:              slugify(e.Summary),
			Location:          e.Location,
			YouTubeLiveURL:    formulaDriftYouTubeLiveURL,
			YouTubeChannelURL: formulaDriftYouTubeChannelURL,
			YouTubeURL:        formulaDriftYouTubeChannelURL,
			Start:             e.Start,
			End:               end,
			Source:            "ics",
		})
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Start.Before(out[j].Start) })
	enrichYouTubeURLs(ctx, f.httpClient, out, time.Now().UTC())
	return out, nil
}

func slugify(s string) string {
	s = strings.ToLower(s)
	var b strings.Builder
	lastDash := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastDash = false
			continue
		}
		if !lastDash {
			b.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(b.String(), "-")
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// Client selects ICS or formulad as the drift schedule source.
type Client struct {
	ics       *ICSFetcher
	formulad  *FormuladFetcher
	preferICS bool
}

func NewClient(source, icsURL, formuladURL string) *Client {
	source = strings.ToLower(strings.TrimSpace(source))
	if source == "" {
		source = "formulad"
	}
	c := &Client{
		formulad:  NewFormuladFetcher(formuladURL),
		preferICS: source == "ics",
	}
	if icsURL != "" {
		c.ics = NewICSFetcher(icsURL)
	}
	return c
}

func (c *Client) FetchEvents(ctx context.Context) ([]Event, error) {
	if c.preferICS && c.ics != nil {
		events, err := c.ics.FetchEvents(ctx)
		if err == nil && len(events) > 0 {
			return events, nil
		}
	}
	if c.formulad != nil {
		return c.formulad.FetchEvents(ctx)
	}
	if c.ics != nil {
		return c.ics.FetchEvents(ctx)
	}
	return nil, fmt.Errorf("drift: no schedule source configured")
}
