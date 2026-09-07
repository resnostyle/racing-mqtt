package racing

import (
	"context"
	"log/slog"
	"time"

	"github.com/resnostyle/mqttkit/mqttpub"
	"github.com/resnostyle/racing-mqtt/internal/lib/vir"
)

// VirFetcher loads VIR schedule events.
type VirFetcher interface {
	FetchEvents(ctx context.Context) ([]vir.Event, error)
}

func VirTopicPrefix(settings Settings) string {
	return settings.MQTTTopicPrefix + "/vir"
}

func PublishVir(ctx context.Context, settings Settings, fetcher VirFetcher, mqtt mqttpub.Sink) ([]vir.Event, error) {
	if !settings.VirEnabled || fetcher == nil {
		return nil, nil
	}
	events, err := fetcher.FetchEvents(ctx)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	active := vir.ActiveEvent(events, now)
	next := vir.NextEvent(events, now)

	var display vir.Event
	live := active != nil
	if live {
		display = *active
	} else if next != nil {
		display = *next
	} else if len(events) > 0 {
		display = events[len(events)-1]
	}

	horizon := time.Duration(settings.UpcomingDays) * 24 * time.Hour
	upcoming := vir.UpcomingEvents(events, now, horizon)

	summary := BuildVirSummaryPayload(live, virDisplayPtr(active, next, live), next, len(upcoming))
	if err := publishScheduleTopics(mqtt, "vir",
		BuildVirActivePayload(live, display),
		BuildVirUpcomingPayload(upcoming),
		summary,
	); err != nil {
		return events, err
	}

	slog.Info("published vir topics",
		"live", live,
		"event", display.Name,
		"series", display.Series,
		"upcoming", len(upcoming),
	)
	return events, nil
}

func virDisplayPtr(active, next *vir.Event, live bool) *vir.Event {
	if live {
		return active
	}
	return next
}

func PublishVirDiscovery(settings Settings, mqtt mqttpub.Sink) error {
	if !settings.MQTTDiscoveryEnabled || !settings.VirEnabled {
		return nil
	}
	configs := BuildVirDiscoveryConfigs(VirTopicPrefix(settings))
	if err := mqtt.PublishDiscovery(configs, settings.MQTTDiscoveryPrefix); err != nil {
		return err
	}
	slog.Info("published vir discovery configs", "count", len(configs))
	return nil
}

func VirPollInterval(settings Settings, events []vir.Event, now time.Time) time.Duration {
	return boundaryAwarePoll(settings.PollIntervalSeconds, vir.NextBoundary(events, now), now)
}
