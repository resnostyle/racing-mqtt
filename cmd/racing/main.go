package main

import (
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/resnostyle/mqttkit/logx"
	"github.com/resnostyle/mqttkit/mqttpub"
	"github.com/resnostyle/mqttkit/poll"
	"github.com/resnostyle/racing-mqtt/internal/lib/drift"
	"github.com/resnostyle/racing-mqtt/internal/lib/lightsouts"
	"github.com/resnostyle/racing-mqtt/internal/lib/openwec"
	"github.com/resnostyle/racing-mqtt/internal/lib/vir"
	"github.com/resnostyle/racing-mqtt/internal/racing"
)

func main() {
	settings, err := racing.FromEnv()
	if err != nil {
		slog.Error(err.Error())
		os.Exit(1)
	}
	logx.Configure(settings.LogLevel, false)

	slog.Info("starting racing-mqtt",
		"series", strings.Join(settings.SeriesSlugs, ","),
		"poll_interval", settings.PollIntervalSeconds,
		"upcoming_days", settings.UpcomingDays,
		"mqtt", settings.MQTTHost,
		"port", settings.MQTTPort,
		"discovery", settings.MQTTDiscoveryEnabled,
		"openwec", settings.OpenWECEnabled,
		"drift", settings.DriftEnabled,
		"drift_source", settings.DriftSource,
		"vir", settings.VirEnabled,
	)

	ctx, cancel := poll.NotifyContext()
	defer cancel()

	mqtt, err := mqttpub.New(
		settings.MQTTHost,
		settings.MQTTPort,
		settings.MQTTClientID,
		settings.MQTTUsername,
		settings.MQTTPassword,
		settings.MQTTTopicPrefix,
	)
	if err != nil {
		slog.Error("mqtt connect failed", "err", err)
		os.Exit(1)
	}
	defer mqtt.Close()

	if err := racing.PublishDiscovery(settings, mqtt); err != nil {
		slog.Error("mqtt discovery publish failed", "err", err)
	}
	if err := racing.PublishDriftDiscovery(settings, mqtt); err != nil {
		slog.Error("drifting discovery publish failed", "err", err)
	}
	if err := racing.PublishVirDiscovery(settings, mqtt); err != nil {
		slog.Error("vir discovery publish failed", "err", err)
	}

	fetcher := lightsouts.NewClient()
	var lastSessions []lightsouts.Session
	var lastDrift []drift.Event
	var lastVir []vir.Event
	var driftFetcher racing.DriftFetcher
	if settings.DriftEnabled {
		driftFetcher = drift.NewClient(settings.DriftSource, settings.DriftICSURL, settings.DriftFormuladURL)
	}
	var virFetcher racing.VirFetcher
	if settings.VirEnabled {
		virFetcher = vir.NewClient(settings.VirEventsURL, settings.VirWPPostsURL, settings.VirWPCategoryID)
	}
	var resultsPub *racing.ResultsPublisher
	if settings.OpenWECEnabled {
		resultsPub = racing.NewResultsPublisher(openwec.NewClient(settings.OpenWECBaseURL, settings.OpenWECAPIKey))
	}

	for ctx.Err() == nil {
		sessions, err := racing.FetchAndPublish(ctx, settings, fetcher, mqtt)
		if err != nil {
			slog.Error("fetch/publish failed", "err", err)
		} else {
			lastSessions = sessions
			if resultsPub != nil {
				if err := resultsPub.PublishFinished(ctx, settings, mqtt, sessions); err != nil {
					slog.Warn("results publish failed", "err", err)
				}
			}
		}
		if driftFetcher != nil {
			events, err := racing.PublishDrift(ctx, settings, driftFetcher, mqtt)
			if err != nil {
				slog.Warn("drifting publish failed", "err", err)
			} else if events != nil {
				lastDrift = events
			}
		}
		if virFetcher != nil {
			events, err := racing.PublishVir(ctx, settings, virFetcher, mqtt)
			if err != nil {
				slog.Warn("vir publish failed", "err", err)
			} else if events != nil {
				lastVir = events
			}
		}
		now := time.Now().UTC()
		wait := racing.PollInterval(settings, lastSessions, now)
		if settings.DriftEnabled {
			wait = racing.MinPollInterval(wait, racing.DriftPollInterval(settings, lastDrift, now))
		}
		if settings.VirEnabled {
			wait = racing.MinPollInterval(wait, racing.VirPollInterval(settings, lastVir, now))
		}
		slog.Debug("waiting for next poll", "seconds", wait.Seconds())
		poll.Wait(ctx, wait)
	}
	slog.Info("exited")
}
