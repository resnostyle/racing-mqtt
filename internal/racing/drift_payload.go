package racing

import (
	"time"

	"github.com/resnostyle/mqttkit/payload"
	"github.com/resnostyle/racing-mqtt/internal/lib/drift"
)

func BuildDriftActivePayload(live bool, e drift.Event) map[string]any {
	out := driftEventFields(e)
	out["live"] = live
	out["published_at"] = payload.UTCNowISO()
	return out
}

func BuildDriftUpcomingPayload(events []drift.Event) map[string]any {
	items := make([]map[string]any, 0, len(events))
	for _, e := range events {
		items = append(items, driftEventFields(e))
	}
	return map[string]any{
		"events":       items,
		"count":        len(items),
		"published_at": payload.UTCNowISO(),
	}
}

func BuildDriftSummaryPayload(live bool, current, next *drift.Event, upcomingCount int) map[string]any {
	out := map[string]any{
		"live":           live,
		"upcoming_count": upcomingCount,
		"published_at":   payload.UTCNowISO(),
	}
	if current != nil {
		out["current"] = driftEventFields(*current)
	}
	if next != nil {
		out["next"] = driftNextFields(*next)
	}
	return out
}

func driftEventFields(e drift.Event) map[string]any {
	return map[string]any{
		"series":              e.Series,
		"name":                e.Name,
		"slug":                e.Slug,
		"round":               e.Round,
		"venue":               payload.NilIfEmpty(e.Venue),
		"location":            payload.NilIfEmpty(e.Location),
		"hashtag":             payload.NilIfEmpty(e.Hashtag),
		"link":                payload.NilIfEmpty(e.Link),
		"youtube_url":         payload.NilIfEmpty(e.YouTubeURL),
		"youtube_live_url":    payload.NilIfEmpty(e.YouTubeLiveURL),
		"youtube_channel_url": payload.NilIfEmpty(e.YouTubeChannelURL),
		"start":               e.Start.UTC().Format(time.RFC3339),
		"end":                 e.End.UTC().Format(time.RFC3339),
		"uid":                 e.UID,
		"source":              e.Source,
	}
}

func driftNextFields(e drift.Event) map[string]any {
	return map[string]any{
		"name":                e.Name,
		"slug":                e.Slug,
		"round":               e.Round,
		"venue":               payload.NilIfEmpty(e.Venue),
		"location":            payload.NilIfEmpty(e.Location),
		"link":                payload.NilIfEmpty(e.Link),
		"youtube_url":         payload.NilIfEmpty(e.YouTubeURL),
		"youtube_live_url":    payload.NilIfEmpty(e.YouTubeLiveURL),
		"youtube_channel_url": payload.NilIfEmpty(e.YouTubeChannelURL),
		"start":               e.Start.UTC().Format(time.RFC3339),
		"end":                 e.End.UTC().Format(time.RFC3339),
	}
}
