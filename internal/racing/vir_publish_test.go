package racing

import (
	"context"
	"testing"
	"time"

	"github.com/resnostyle/racing-mqtt/internal/lib/vir"
)

type mockVirFetcher struct {
	events []vir.Event
}

func (m *mockVirFetcher) FetchEvents(context.Context) ([]vir.Event, error) {
	return m.events, nil
}

func TestPublishVir(t *testing.T) {
	settings := Settings{VirEnabled: true, UpcomingDays: 90}
	settings.MQTTTopicPrefix = "home/racing"
	start := time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC)
	fetcher := &mockVirFetcher{events: []vir.Event{{
		UID: "vir:2026:imsa", Series: "IMSA", Name: "IMSA Michelin GT Challenge",
		Slug: "imsa-michelin-gt-challenge", Venue: vir.Venue, Location: vir.Location,
		Start: start, End: start.Add(72 * time.Hour), Source: "virnow",
	}}}
	sink := &mockSink{topics: map[string]any{}}
	_, err := PublishVir(context.Background(), settings, fetcher, sink)
	if err != nil {
		t.Fatal(err)
	}
	if sink.topics["vir/summary"] == nil {
		t.Fatal("missing vir summary")
	}
}
