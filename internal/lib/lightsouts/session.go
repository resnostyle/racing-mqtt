package lightsouts

import (
	"fmt"
	"strings"
	"time"
)

const (
	CategoryPractice   = "practice"
	CategoryQualifying = "qualifying"
	CategorySprint     = "sprint"
	CategoryRace       = "race"
	CategoryOther      = "other"

	AllDayThresholdMinutes = 25 * 60
)

// Session is a single motorsport session from the Lightsouts API.
type Session struct {
	UID         string
	SeriesSlug  string
	SeriesName  string
	SeriesShort string
	EventName   string
	EventSlug   string
	Circuit     string
	Country     string
	SessionName string
	Start       time.Time
	End         time.Time
	IsMain      bool
	Category    string
}

// ClassifySession buckets a session name into practice, qualifying, sprint, race, or other.
func ClassifySession(name string) string {
	n := strings.TrimSpace(strings.ToLower(name))
	if n == "sprint" || n == "sprint race" || n == "superpole race" {
		return CategorySprint
	}
	if strings.Contains(n, "practice") || strings.Contains(n, "warm up") {
		return CategoryPractice
	}
	if strings.Contains(n, "qualifying") || strings.Contains(n, "qualifications") ||
		strings.Contains(n, "superpole") || strings.Contains(n, "hyperpole") ||
		strings.Contains(n, "shootout") {
		return CategoryQualifying
	}
	if strings.Contains(n, "sprint") {
		return CategorySprint
	}
	if strings.Contains(n, "race") || strings.Contains(n, "rally") {
		return CategoryRace
	}
	return CategoryOther
}

func parseSessionStart(dateStr, timeStr string) (time.Time, bool) {
	dateStr = strings.TrimSpace(dateStr)
	if dateStr == "" {
		return time.Time{}, false
	}
	if strings.Contains(dateStr, "T") {
		t, err := time.Parse(time.RFC3339, strings.Replace(dateStr, "Z", "+00:00", 1))
		if err != nil {
			return time.Time{}, false
		}
		return t.UTC(), true
	}
	timeStr = strings.TrimSpace(timeStr)
	if timeStr == "" {
		timeStr = "00:00"
	}
	t, err := time.Parse("2006-01-02T15:04:05Z07:00", dateStr+"T"+timeStr+":00+00:00")
	if err != nil {
		return time.Time{}, false
	}
	return t.UTC(), true
}

// FlattenSeries converts a series detail API payload into sessions.
func FlattenSeries(payload map[string]any) []Session {
	seriesName, _ := payload["name"].(string)
	if seriesName == "" {
		seriesName, _ = payload["slug"].(string)
	}
	seriesShort, _ := payload["nameAlt"].(string)
	if seriesShort == "" {
		seriesShort = seriesName
	}
	seriesSlug, _ := payload["slug"].(string)

	events, _ := payload["events"].([]any)
	out := make([]Session, 0)
	for _, rawEvent := range events {
		event, ok := rawEvent.(map[string]any)
		if !ok {
			continue
		}
		if tba, _ := event["tba"].(bool); tba {
			continue
		}
		eventName, _ := event["name"].(string)
		eventSlug, _ := event["slug"].(string)
		circuit, country := eventLocation(event)

		sessions, _ := event["sessions"].([]any)
		for _, rawSession := range sessions {
			s, ok := rawSession.(map[string]any)
			if !ok {
				continue
			}
			dateStr, _ := s["date"].(string)
			timeStr, _ := s["time"].(string)
			start, ok := parseSessionStart(dateStr, timeStr)
			if !ok {
				continue
			}
			duration := 60
			if d, ok := s["duration"].(float64); ok {
				duration = int(d)
			}
			end := start.Add(time.Duration(duration) * time.Minute)
			sessionName, _ := s["name"].(string)
			if sessionName == "" {
				sessionName = "Session"
			}
			sid := sessionID(s["id"])
			uid := seriesSlug + ":" + eventSlug + ":" + sid
			if sid == "" {
				uid = seriesSlug + ":" + eventSlug + ":" + sessionName + ":" + start.Format(time.RFC3339)
			}
			out = append(out, Session{
				UID:         uid,
				SeriesSlug:  seriesSlug,
				SeriesName:  seriesName,
				SeriesShort: seriesShort,
				EventName:   eventName,
				EventSlug:   eventSlug,
				Circuit:     circuit,
				Country:     country,
				SessionName: sessionName,
				Start:       start,
				End:         end,
				IsMain:      boolVal(s["main"]),
				Category:    ClassifySession(sessionName),
			})
		}
	}
	return out
}

func sessionID(v any) string {
	switch n := v.(type) {
	case float64:
		return fmt.Sprintf("%.0f", n)
	case int:
		return fmt.Sprintf("%d", n)
	case int64:
		return fmt.Sprintf("%d", n)
	default:
		return ""
	}
}

func boolVal(v any) bool {
	b, _ := v.(bool)
	return b
}

func eventLocation(event map[string]any) (circuit, country string) {
	if c, ok := event["circuit"].(map[string]any); ok {
		circuit, _ = c["name"].(string)
		if co, ok := c["country"].(map[string]any); ok {
			country, _ = co["name"].(string)
		}
	}
	if country == "" {
		if co, ok := event["country"].(map[string]any); ok {
			country, _ = co["name"].(string)
		}
	}
	return circuit, country
}
