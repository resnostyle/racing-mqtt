package racing

import (
	"context"
	"testing"
	"time"

	"github.com/resnostyle/racing-mqtt/internal/lib/drift"
)

type mockDriftFetcher struct {
	events []drift.Event
}

func (m *mockDriftFetcher) FetchEvents(context.Context) ([]drift.Event, error) {
	return m.events, nil
}

func TestPublishDrift(t *testing.T) {
	settings := Settings{DriftEnabled: true, UpcomingDays: 90}
	settings.MQTTTopicPrefix = "home/racing"
	start := time.Date(2026, 10, 23, 14, 0, 0, 0, time.UTC)
	fetcher := &mockDriftFetcher{events: []drift.Event{{
		UID: "fd:1", Series: "Formula Drift", Name: "SHORELINE SHOWDOWN", Slug: "long-beach2",
		Round: 8, Start: start, End: start.Add(48 * time.Hour), Source: "formulad",
	}}}
	sink := &mockSink{topics: map[string]any{}}
	_, err := PublishDrift(context.Background(), settings, fetcher, sink)
	if err != nil {
		t.Fatal(err)
	}
	if sink.topics["drifting/summary"] == nil {
		t.Fatal("missing drifting summary")
	}
}
