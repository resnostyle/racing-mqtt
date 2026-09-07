package racing

import (
	"context"
	"log/slog"
	"time"

	"github.com/resnostyle/mqttkit/mqttpub"
	"github.com/resnostyle/racing-mqtt/internal/lib/drift"
)

// DriftFetcher loads Formula Drift schedule events.
type DriftFetcher interface {
	FetchEvents(ctx context.Context) ([]drift.Event, error)
}

func DriftTopicPrefix(settings Settings) string {
	return settings.MQTTTopicPrefix + "/drifting"
}

func PublishDrift(ctx context.Context, settings Settings, fetcher DriftFetcher, mqtt mqttpub.Sink) ([]drift.Event, error) {
	if !settings.DriftEnabled || fetcher == nil {
		return nil, nil
	}
	events, err := fetcher.FetchEvents(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	active := drift.ActiveEvent(events, now)
	next := drift.NextEvent(events, now)

	var display drift.Event
	live := active != nil
	if live {
		display = *active
	} else if next != nil {
		display = *next
	} else if len(events) > 0 {
		display = events[len(events)-1]
	}

	horizon := time.Duration(settings.UpcomingDays) * 24 * time.Hour
	upcoming := drift.UpcomingEvents(events, now, horizon)

	summary := BuildDriftSummaryPayload(live, driftDisplayPtr(active, next, live), next, len(upcoming))
	if err := publishScheduleTopics(mqtt, "drifting",
		BuildDriftActivePayload(live, display),
		BuildDriftUpcomingPayload(upcoming),
		summary,
	); err != nil {
		return events, err
	}

	slog.Info("published drifting topics",
		"live", live,
		"event", display.Name,
		"round", display.Round,
		"upcoming", len(upcoming),
	)
	return events, nil
}

func driftDisplayPtr(active, next *drift.Event, live bool) *drift.Event {
	if live {
		return active
	}
	return next
}

func PublishDriftDiscovery(settings Settings, mqtt mqttpub.Sink) error {
	if !settings.MQTTDiscoveryEnabled || !settings.DriftEnabled {
		return nil
	}
	configs := BuildDriftDiscoveryConfigs(DriftTopicPrefix(settings))
	if err := mqtt.PublishDiscovery(configs, settings.MQTTDiscoveryPrefix); err != nil {
		return err
	}
	slog.Info("published drifting discovery configs", "count", len(configs))
	return nil
}

// DriftPollInterval returns wait duration based on drift event boundaries.
func DriftPollInterval(settings Settings, events []drift.Event, now time.Time) time.Duration {
	return boundaryAwarePoll(settings.PollIntervalSeconds, drift.NextBoundary(events, now), now)
}

// MinPollInterval returns the shorter of two poll waits.
func MinPollInterval(enduranceWait time.Duration, driftWait time.Duration) time.Duration {
	if driftWait > 0 && driftWait < enduranceWait {
		return driftWait
	}
	return enduranceWait
}
