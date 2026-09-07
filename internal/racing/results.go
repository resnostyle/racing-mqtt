package racing

import (
	"context"
	"log/slog"
	"strings"
	"time"

	"github.com/resnostyle/mqttkit/mqttpub"
	"github.com/resnostyle/racing-mqtt/internal/lib/lightsouts"
	"github.com/resnostyle/racing-mqtt/internal/lib/openwec"
)

// ResultsClient fetches OpenWEC event/session/result data.
type ResultsClient interface {
	ListEvents(ctx context.Context, seriesKey string, year int) ([]openwec.Event, error)
	ListSessions(ctx context.Context, seriesKey string, year, eventID int) ([]openwec.Session, error)
	GetResults(ctx context.Context, sessionID int) ([]openwec.Result, error)
}

// ResultsPublisher resolves Lightsouts races to OpenWEC results and publishes MQTT payloads.
type ResultsPublisher struct {
	client    ResultsClient
	published map[string]struct{}
}

func NewResultsPublisher(client ResultsClient) *ResultsPublisher {
	return &ResultsPublisher{
		client:    client,
		published: make(map[string]struct{}),
	}
}

func (p *ResultsPublisher) PublishFinished(ctx context.Context, settings Settings, mqtt mqttpub.Sink, sessions []lightsouts.Session) error {
	if !settings.OpenWECEnabled || p.client == nil {
		return nil
	}
	now := time.Now().UTC()
	lookback := time.Duration(settings.OpenWECResultsLookbackDays) * 24 * time.Hour
	var firstErr error

	for _, lo := range sessions {
		if lo.Category != lightsouts.CategoryRace {
			continue
		}
		if !lo.End.Before(now) {
			continue
		}
		if now.Sub(lo.End) > lookback {
			continue
		}
		if _, done := p.published[lo.UID]; done {
			continue
		}

		seriesKey, ok := openwec.SeriesKeyForLightsouts(lo.SeriesSlug)
		if !ok {
			continue
		}

		year := lo.Start.Year()
		events, err := p.client.ListEvents(ctx, seriesKey, year)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			slog.Warn("openwec list events failed", "series", seriesKey, "year", year, "err", err)
			continue
		}

		event, eventScore := openwec.MatchEvent(lo, events)
		if event == nil || eventScore < openwec.MinEventMatchScore() {
			slog.Debug("openwec event not matched", "uid", lo.UID, "event", lo.EventName, "circuit", lo.Circuit)
			continue
		}

		owSessions, err := p.client.ListSessions(ctx, seriesKey, year, event.ID)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			slog.Warn("openwec list sessions failed", "event", event.Name, "err", err)
			continue
		}

		owSession, _, score := openwec.Match(lo, events, owSessions)
		if owSession == nil {
			slog.Debug("openwec session not matched", "uid", lo.UID, "event", event.Name)
			continue
		}

		results, err := p.client.GetResults(ctx, owSession.ID)
		if err != nil {
			if firstErr == nil {
				firstErr = err
			}
			slog.Warn("openwec results not available yet", "session_id", owSession.ID, "err", err)
			continue
		}
		if len(results) == 0 {
			continue
		}

		payload := BuildResultsPayload(lo, seriesKey, event, owSession, results)
		suffix := "results/" + strings.ToLower(seriesKey) + "/current"
		if err := mqtt.Publish(suffix, payload, true); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}
		eventSuffix := "results/" + strings.ToLower(seriesKey) + "/" + slugify(lo.EventSlug)
		if err := mqtt.Publish(eventSuffix, payload, true); err != nil {
			if firstErr == nil {
				firstErr = err
			}
			continue
		}

		p.published[lo.UID] = struct{}{}
		slog.Info("published race results",
			"series", seriesKey,
			"event", lo.EventName,
			"openwec_session", owSession.ID,
			"match_score", score,
			"entries", len(dedupeResults(results)),
		)
	}
	return firstErr
}

func slugify(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
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
