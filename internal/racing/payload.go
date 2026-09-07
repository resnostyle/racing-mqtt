package racing

import (
	"strings"

	"github.com/resnostyle/mqttkit/payload"
	"github.com/resnostyle/racing-mqtt/internal/lib/lightsouts"
)

func sessionLocation(s lightsouts.Session) string {
	parts := make([]string, 0, 2)
	if s.Circuit != "" {
		parts = append(parts, s.Circuit)
	}
	if s.Country != "" {
		parts = append(parts, s.Country)
	}
	return strings.Join(parts, ", ")
}

func sessionFields(s lightsouts.Session) map[string]any {
	return map[string]any{
		"series":      s.SeriesShort,
		"series_full": s.SeriesName,
		"series_slug": s.SeriesSlug,
		"event":       s.EventName,
		"event_slug":  s.EventSlug,
		"session":     s.SessionName,
		"circuit":     payload.NilIfEmpty(s.Circuit),
		"country":     payload.NilIfEmpty(s.Country),
		"category":    s.Category,
		"start":       s.Start.UTC().Format("2006-01-02T15:04:05Z07:00"),
		"end":         s.End.UTC().Format("2006-01-02T15:04:05Z07:00"),
		"uid":         s.UID,
		"is_main":     s.IsMain,
	}
}

func nextSessionFields(s lightsouts.Session) map[string]any {
	return map[string]any{
		"series":      s.SeriesShort,
		"series_slug": s.SeriesSlug,
		"event":       s.EventName,
		"session":     s.SessionName,
		"circuit":     payload.NilIfEmpty(s.Circuit),
		"country":     payload.NilIfEmpty(s.Country),
		"location":    payload.NilIfEmpty(sessionLocation(s)),
		"start":       s.Start.UTC().Format("2006-01-02T15:04:05Z07:00"),
	}
}

func BuildActivePayload(live bool, session lightsouts.Session) map[string]any {
	p := sessionFields(session)
	p["live"] = live
	p["published_at"] = payload.UTCNowISO()
	return p
}

func BuildUpcomingPayload(sessions []lightsouts.Session) map[string]any {
	items := make([]map[string]any, 0, len(sessions))
	for _, s := range sessions {
		items = append(items, sessionFields(s))
	}
	return map[string]any{
		"sessions":     items,
		"count":        len(items),
		"published_at": payload.UTCNowISO(),
	}
}

func BuildSummaryPayload(
	live bool,
	current *lightsouts.Session,
	nextBySeries map[string]lightsouts.Session,
	upcomingCount int,
) map[string]any {
	out := map[string]any{
		"live":           live,
		"upcoming_count": upcomingCount,
		"published_at":   payload.UTCNowISO(),
		"next_by_series": map[string]any{},
	}
	if current != nil {
		out["current"] = sessionFields(*current)
	}
	seriesMap := make(map[string]any, len(nextBySeries))
	for key, s := range nextBySeries {
		seriesMap[key] = nextSessionFields(s)
	}
	out["next_by_series"] = seriesMap
	return out
}
