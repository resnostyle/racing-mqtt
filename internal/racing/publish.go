package racing

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/resnostyle/mqttkit/mqttpub"
	"github.com/resnostyle/racing-mqtt/internal/lib/lightsouts"
)

// Fetcher loads sessions from external sources.
type Fetcher interface {
	FetchSeriesSlugs(ctx context.Context, slugs []string) ([]lightsouts.Session, error)
}

func PublishDiscovery(settings Settings, mqtt mqttpub.Sink) error {
	if !settings.MQTTDiscoveryEnabled {
		slog.Info("MQTT discovery disabled")
		return nil
	}
	configs := BuildDiscoveryConfigs(settings.MQTTTopicPrefix, settings.SeriesSlugs)
	if err := mqtt.PublishDiscovery(configs, settings.MQTTDiscoveryPrefix); err != nil {
		return err
	}
	slog.Info("published mqtt discovery configs", "count", len(configs))
	return nil
}

func FetchAndPublish(ctx context.Context, settings Settings, fetcher Fetcher, mqtt mqttpub.Sink) ([]lightsouts.Session, error) {
	raw, err := fetcher.FetchSeriesSlugs(ctx, settings.SeriesSlugs)
	if err != nil {
		return nil, err
	}
	sessions := settings.FilterSessions(raw)
	sortSessions(sessions)

	now := time.Now().UTC()
	active := ActiveSession(sessions, now)
	next := NextSession(sessions, now)

	var display lightsouts.Session
	live := active != nil
	if live {
		display = *active
	} else if next != nil {
		display = *next
	} else if len(sessions) > 0 {
		display = sessions[len(sessions)-1]
	}

	horizon := time.Duration(settings.UpcomingDays) * 24 * time.Hour
	upcoming := UpcomingSessions(sessions, now, horizon)
	nextBySeries := NextBySeries(sessions, now)

	if err := mqtt.Publish("active/current", BuildActivePayload(live, display), true); err != nil {
		return sessions, fmt.Errorf("publish active: %w", err)
	}
	if err := mqtt.Publish("upcoming/current", BuildUpcomingPayload(upcoming), true); err != nil {
		return sessions, fmt.Errorf("publish upcoming: %w", err)
	}
	summary := BuildSummaryPayload(live, sessionPtr(active, next, live), nextBySeries, len(upcoming))
	if err := mqtt.Publish("summary", summary, true); err != nil {
		return sessions, fmt.Errorf("publish summary: %w", err)
	}

	slog.Info("published racing topics",
		"live", live,
		"series", display.SeriesShort,
		"session", display.SessionName,
		"upcoming", len(upcoming),
	)
	return sessions, nil
}

func sessionPtr(active, next *lightsouts.Session, live bool) *lightsouts.Session {
	if live {
		return active
	}
	return next
}

func sortSessions(sessions []lightsouts.Session) {
	for i := 0; i < len(sessions); i++ {
		for j := i + 1; j < len(sessions); j++ {
			if sessions[j].Start.Before(sessions[i].Start) {
				sessions[i], sessions[j] = sessions[j], sessions[i]
			}
		}
	}
}
