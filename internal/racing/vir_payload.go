package racing

import (
	"time"

	"github.com/resnostyle/mqttkit/payload"
	"github.com/resnostyle/racing-mqtt/internal/lib/vir"
)

func BuildVirActivePayload(live bool, e vir.Event) map[string]any {
	out := virEventFields(e)
	out["live"] = live
	out["published_at"] = payload.UTCNowISO()
	return out
}

func BuildVirUpcomingPayload(events []vir.Event) map[string]any {
	items := make([]map[string]any, 0, len(events))
	for _, e := range events {
		items = append(items, virEventFields(e))
	}
	return map[string]any{
		"events":       items,
		"count":        len(items),
		"published_at": payload.UTCNowISO(),
	}
}

func BuildVirSummaryPayload(live bool, current, next *vir.Event, upcomingCount int) map[string]any {
	out := map[string]any{
		"live":           live,
		"upcoming_count": upcomingCount,
		"published_at":   payload.UTCNowISO(),
	}
	if current != nil {
		out["current"] = virEventFields(*current)
	}
	if next != nil {
		out["next"] = virNextFields(*next)
	}
	return out
}

func virEventFields(e vir.Event) map[string]any {
	return map[string]any{
		"series":   e.Series,
		"name":     e.Name,
		"slug":     e.Slug,
		"venue":    payload.NilIfEmpty(e.Venue),
		"location": payload.NilIfEmpty(e.Location),
		"link":     payload.NilIfEmpty(e.Link),
		"start":    e.Start.UTC().Format(time.RFC3339),
		"end":      e.End.UTC().Format(time.RFC3339),
		"uid":      e.UID,
		"source":   e.Source,
	}
}

func virNextFields(e vir.Event) map[string]any {
	return map[string]any{
		"series":   e.Series,
		"name":     e.Name,
		"slug":     e.Slug,
		"venue":    payload.NilIfEmpty(e.Venue),
		"location": payload.NilIfEmpty(e.Location),
		"link":     payload.NilIfEmpty(e.Link),
		"start":    e.Start.UTC().Format(time.RFC3339),
	}
}
