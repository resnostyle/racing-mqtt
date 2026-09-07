package vir

import (
	"context"
	"encoding/json"
	"fmt"
	"html"
	"io"
	"net/http"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"
)

const (
	defaultEventsURL  = "https://virnow.com/events/"
	defaultWPPostsURL = "https://virnow.com/wp-json/wp/v2/posts"
	defaultUserAgent  = "racing-mqtt/1.0 (https://github.com/resnostyle/racing-mqtt)"
	requestTimeout    = 30 * time.Second
	virTZ             = "America/New_York"
)

var (
	cardTitleRE     = regexp.MustCompile(`<h2 class="ha-card-title">([^<]+)</h2>`)
	cardTitleTextRE = regexp.MustCompile(`^(.+?):\s*([A-Za-z]+)\s+(\d{1,2})(?:-(\d{1,2}))?,?\s*(\d{4})$`)
	excerptDateRE   = regexp.MustCompile(`(?i)([A-Za-z]+)\s+(\d{1,2})(?:-(\d{1,2}))?(?:,?\s*(\d{4}))?`)
)

var monthMap = map[string]time.Month{
	"jan": time.January, "january": time.January,
	"feb": time.February, "february": time.February,
	"mar": time.March, "march": time.March,
	"apr": time.April, "april": time.April,
	"may": time.May,
	"jun": time.June, "june": time.June,
	"jul": time.July, "july": time.July,
	"aug": time.August, "august": time.August,
	"sep": time.September, "sept": time.September, "september": time.September,
	"oct": time.October, "october": time.October,
	"nov": time.November, "november": time.November,
	"dec": time.December, "december": time.December,
}

// Client loads VIR events from virnow.com.
type Client struct {
	httpClient *http.Client
	eventsURL  string
	wpPostsURL string
	wpCategory int
}

func NewClient(eventsURL, wpPostsURL string, wpCategory int) *Client {
	if strings.TrimSpace(eventsURL) == "" {
		eventsURL = defaultEventsURL
	}
	if strings.TrimSpace(wpPostsURL) == "" {
		wpPostsURL = defaultWPPostsURL
	}
	if wpCategory <= 0 {
		wpCategory = 81
	}
	return &Client{
		httpClient: &http.Client{Timeout: requestTimeout},
		eventsURL:  eventsURL,
		wpPostsURL: wpPostsURL,
		wpCategory: wpCategory,
	}
}

func (c *Client) FetchEvents(ctx context.Context) ([]Event, error) {
	body, err := c.get(ctx, c.eventsURL)
	if err != nil {
		return nil, err
	}
	events, err := parseEventsPage(string(body))
	if err != nil {
		return nil, err
	}
	if err := c.enrichLinks(ctx, events); err != nil {
		// Links are optional; keep schedule if WP API is unavailable.
		_ = err
	}
	sort.Slice(events, func(i, j int) bool { return events[i].Start.Before(events[j].Start) })
	return events, nil
}

func parseEventsPage(pageHTML string) ([]Event, error) {
	loc, err := time.LoadLocation(virTZ)
	if err != nil {
		return nil, fmt.Errorf("vir: load timezone: %w", err)
	}
	titles := cardTitleRE.FindAllStringSubmatch(pageHTML, -1)
	if len(titles) == 0 {
		return nil, fmt.Errorf("vir: no events found on page")
	}
	events := make([]Event, 0, len(titles))
	for _, m := range titles {
		title := html.UnescapeString(strings.TrimSpace(m[1]))
		name, start, end, ok := parseCardTitle(title)
		if !ok {
			continue
		}
		slug := slugify(name)
		events = append(events, Event{
			UID:      fmt.Sprintf("vir:%d:%s", start.Year(), slug),
			Series:   seriesFromName(name),
			Name:     name,
			Slug:     slug,
			Location: Location,
			Venue:    Venue,
			Start:    start.In(loc).UTC(),
			End:      end.In(loc).UTC(),
			Source:   "virnow",
		})
	}
	if len(events) == 0 {
		return nil, fmt.Errorf("vir: no events parsed")
	}
	return events, nil
}

func parseCardTitle(title string) (name string, start, end time.Time, ok bool) {
	m := cardTitleTextRE.FindStringSubmatch(title)
	if len(m) != 6 {
		return "", time.Time{}, time.Time{}, false
	}
	name = strings.TrimSpace(m[1])
	month, okMonth := monthMap[strings.ToLower(m[2])]
	if !okMonth {
		return "", time.Time{}, time.Time{}, false
	}
	startDay, err := strconv.Atoi(m[3])
	if err != nil {
		return "", time.Time{}, time.Time{}, false
	}
	endDay := startDay
	if m[4] != "" {
		endDay, err = strconv.Atoi(m[4])
		if err != nil {
			return "", time.Time{}, time.Time{}, false
		}
	}
	year, err := strconv.Atoi(m[5])
	if err != nil {
		return "", time.Time{}, time.Time{}, false
	}
	loc, err := time.LoadLocation(virTZ)
	if err != nil {
		return "", time.Time{}, time.Time{}, false
	}
	start = time.Date(year, month, startDay, 8, 0, 0, 0, loc)
	end = time.Date(year, month, endDay, 23, 59, 0, 0, loc).Add(time.Minute)
	return name, start, end, true
}

func seriesFromName(name string) string {
	upper := strings.ToUpper(name)
	switch {
	case strings.Contains(upper, "IMSA"):
		return "IMSA"
	case strings.Contains(upper, "RACING AMERICA"):
		return "Racing America"
	case strings.Contains(upper, "VETERANS"):
		return "Veterans Race of Remembrance"
	case strings.Contains(upper, "CHARITY LAPS"):
		return "Charity Laps"
	default:
		return "VIR"
	}
}

type wpPost struct {
	Link    string `json:"link"`
	Excerpt struct {
		Rendered string `json:"rendered"`
	} `json:"excerpt"`
}

func (c *Client) enrichLinks(ctx context.Context, events []Event) error {
	url := fmt.Sprintf("%s?categories=%d&per_page=100&_fields=link,excerpt", c.wpPostsURL, c.wpCategory)
	body, err := c.get(ctx, url)
	if err != nil {
		return err
	}
	var posts []wpPost
	if err := json.Unmarshal(body, &posts); err != nil {
		return err
	}
	byStart := make(map[string]string, len(posts))
	for _, post := range posts {
		text := html.UnescapeString(stripTags(post.Excerpt.Rendered))
		start, ok := parseLooseDate(text, 0)
		if !ok {
			continue
		}
		key := start.Format("2006-01-02")
		if _, exists := byStart[key]; !exists {
			byStart[key] = post.Link
		}
	}
	for i := range events {
		key := events[i].Start.In(mustLocation()).Format("2006-01-02")
		if link, ok := byStart[key]; ok {
			events[i].Link = link
		}
	}
	return nil
}

func parseLooseDate(text string, defaultYear int) (time.Time, bool) {
	m := excerptDateRE.FindStringSubmatch(text)
	if len(m) < 3 {
		return time.Time{}, false
	}
	month, ok := monthMap[strings.ToLower(m[1])]
	if !ok {
		return time.Time{}, false
	}
	startDay, err := strconv.Atoi(m[2])
	if err != nil {
		return time.Time{}, false
	}
	year := defaultYear
	if len(m) > 4 && m[4] != "" {
		year, err = strconv.Atoi(m[4])
		if err != nil {
			return time.Time{}, false
		}
	}
	if year == 0 {
		year = time.Now().Year()
	}
	loc := mustLocation()
	return time.Date(year, month, startDay, 8, 0, 0, 0, loc), true
}

func mustLocation() *time.Location {
	loc, err := time.LoadLocation(virTZ)
	if err != nil {
		return time.UTC
	}
	return loc
}

func stripTags(s string) string {
	s = regexp.MustCompile(`<[^>]+>`).ReplaceAllString(s, " ")
	return strings.Join(strings.Fields(s), " ")
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

func (c *Client) get(ctx context.Context, url string) ([]byte, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", defaultUserAgent)
	req.Header.Set("Accept", "text/html,application/json")
	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return nil, fmt.Errorf("vir HTTP %d for %s", resp.StatusCode, url)
	}
	return io.ReadAll(resp.Body)
}
