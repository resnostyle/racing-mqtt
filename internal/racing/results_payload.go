package racing

import (
	"fmt"
	"strings"
	"time"

	"github.com/resnostyle/mqttkit/payload"
	"github.com/resnostyle/racing-mqtt/internal/lib/lightsouts"
	"github.com/resnostyle/racing-mqtt/internal/lib/openwec"
)

func BuildResultsPayload(
	lo lightsouts.Session,
	seriesKey string,
	event *openwec.Event,
	owSession *openwec.Session,
	raw []openwec.Result,
) map[string]any {
	byClass := podiumByClass(dedupeResults(raw))
	podium := make(map[string]any, len(byClass))
	for class, rows := range byClass {
		items := make([]map[string]any, 0, len(rows))
		for _, r := range rows {
			items = append(items, resultRow(r))
		}
		podium[class] = items
	}

	sessionAt := owSession.SessionAt
	if t, err := owSession.ParsedSessionAt(); err == nil {
		sessionAt = t.UTC().Format(time.RFC3339)
	}

	return map[string]any{
		"series":             seriesKey,
		"series_slug":        lo.SeriesSlug,
		"event":              lo.EventName,
		"event_slug":         lo.EventSlug,
		"openwec_event":      event.Name,
		"openwec_event_id":   event.ID,
		"openwec_session_id": owSession.ID,
		"lightsouts_uid":     lo.UID,
		"circuit":            payload.NilIfEmpty(lo.Circuit),
		"session_at":         sessionAt,
		"race_start":         lo.Start.UTC().Format(time.RFC3339),
		"race_end":           lo.End.UTC().Format(time.RFC3339),
		"podium_by_class":    podium,
		"entry_count":        len(dedupeResults(raw)),
		"published_at":       payload.UTCNowISO(),
	}
}

func dedupeResults(raw []openwec.Result) []openwec.Result {
	seen := map[string]struct{}{}
	out := make([]openwec.Result, 0, len(raw))
	for _, r := range raw {
		key := r.CarClass + ":" + r.CarNumber
		if _, ok := seen[key]; ok {
			continue
		}
		seen[key] = struct{}{}
		out = append(out, r)
	}
	return out
}

func podiumByClass(results []openwec.Result) map[string][]openwec.Result {
	byClass := make(map[string][]openwec.Result)
	for _, r := range results {
		if r.Position > 3 {
			continue
		}
		byClass[r.CarClass] = append(byClass[r.CarClass], r)
	}
	return byClass
}

func resultRow(r openwec.Result) map[string]any {
	drivers := make([]string, 0, len(r.Drivers))
	for _, d := range r.Drivers {
		name := strings.TrimSpace(d.FirstName + " " + d.LastName)
		if name != "" {
			drivers = append(drivers, name)
		}
	}
	row := map[string]any{
		"position":       r.Position,
		"car_number":     r.CarNumber,
		"car_class":      r.CarClass,
		"team":           r.Team,
		"vehicle":        payload.NilIfEmpty(r.Vehicle),
		"status":         r.Status,
		"laps_completed": r.LapsComplete,
		"drivers":        drivers,
	}
	if r.GapToFirstS != nil {
		row["gap_to_first_s"] = *r.GapToFirstS
	}
	if r.FLTimeS != nil {
		row["fastest_lap_s"] = *r.FLTimeS
	}
	return row
}

// FormatWinnerMessage builds a short notify-friendly summary for the race winner per class.
func FormatWinnerMessage(payload map[string]any) string {
	podium, _ := payload["podium_by_class"].(map[string]any)
	if len(podium) == 0 {
		return ""
	}
	var parts []string
	for class, raw := range podium {
		rows, _ := raw.([]map[string]any)
		if len(rows) == 0 {
			continue
		}
		winner := rows[0]
		team, _ := winner["team"].(string)
		car, _ := winner["car_number"].(string)
		parts = append(parts, fmt.Sprintf("%s: #%s %s", class, car, team))
	}
	event, _ := payload["event"].(string)
	return strings.TrimSpace(event + " — " + strings.Join(parts, "; "))
}
