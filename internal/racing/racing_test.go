package racing

import (
	"context"
	"testing"
	"time"

	"github.com/resnostyle/mqttkit/mqttpub"
	"github.com/resnostyle/racing-mqtt/internal/lib/lightsouts"
)

func sampleSessions() []lightsouts.Session {
	start := time.Date(2026, 9, 6, 18, 0, 0, 0, time.UTC)
	return []lightsouts.Session{
		{
			UID:         "wec:event:1",
			SeriesShort: "WEC",
			SeriesName:  "WEC",
			EventName:   "Lone Star Le Mans",
			SessionName: "Qualifying",
			Start:       start.Add(-24 * time.Hour),
			End:         start.Add(-23 * time.Hour),
			Category:    lightsouts.CategoryQualifying,
		},
		{
			UID:         "wec:event:2",
			SeriesShort: "WEC",
			SeriesName:  "WEC",
			EventName:   "Lone Star Le Mans",
			SessionName: "Race",
			Start:       start,
			End:         start.Add(6 * time.Hour),
			Category:    lightsouts.CategoryRace,
			IsMain:      true,
		},
		{
			UID:         "imsa:event:1",
			SeriesShort: "IMSA",
			SeriesName:  "IMSA SportsCar Championship",
			EventName:   "Battle On The Bricks",
			SessionName: "Race",
			Start:       start.Add(14 * 24 * time.Hour),
			End:         start.Add(14*24*time.Hour + 6*time.Hour),
			Category:    lightsouts.CategoryRace,
			IsMain:      true,
		},
	}
}

func TestActiveAndNextSession(t *testing.T) {
	sessions := sampleSessions()
	now := time.Date(2026, 9, 6, 19, 0, 0, 0, time.UTC)
	active := ActiveSession(sessions, now)
	if active == nil || active.SessionName != "Race" || active.SeriesShort != "WEC" {
		t.Fatalf("active %+v", active)
	}
	next := NextSession(sessions, now)
	if next == nil || next.SeriesShort != "IMSA" {
		t.Fatalf("next %+v", next)
	}
}

func TestNextBySeries(t *testing.T) {
	sessions := sampleSessions()
	now := time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC)
	bySeries := NextBySeries(sessions, now)
	if len(bySeries) != 2 {
		t.Fatalf("got %d entries", len(bySeries))
	}
	if bySeries["WEC"].SessionName != "Qualifying" {
		t.Fatalf("WEC next %+v", bySeries["WEC"])
	}
}

func TestPollIntervalNearBoundary(t *testing.T) {
	settings := Settings{PollIntervalSeconds: 900}
	sessions := sampleSessions()
	now := time.Date(2026, 9, 6, 17, 56, 0, 0, time.UTC)
	wait := PollInterval(settings, sessions, now)
	if wait != nearBoundaryPoll {
		t.Fatalf("got %v want %v", wait, nearBoundaryPoll)
	}
}

func TestFilterSessions(t *testing.T) {
	settings := Settings{Categories: map[string]struct{}{lightsouts.CategoryRace: {}}}
	filtered := settings.FilterSessions(sampleSessions())
	if len(filtered) != 2 {
		t.Fatalf("got %d", len(filtered))
	}
}

type mockFetcher struct {
	sessions []lightsouts.Session
}

func (m *mockFetcher) FetchSeriesSlugs(context.Context, []string) ([]lightsouts.Session, error) {
	return m.sessions, nil
}

type mockSink struct {
	topics map[string]any
}

func (m *mockSink) Publish(suffix string, payload any, retain bool) error {
	m.topics[suffix] = payload
	return nil
}

func (m *mockSink) PublishQuiet(suffix string, payload any, retain bool) error {
	return m.Publish(suffix, payload, retain)
}

func (m *mockSink) PublishRaw(topic string, payload any, retain bool) error {
	return nil
}

func (m *mockSink) PublishDiscovery([]mqttpub.Config, string) error { return nil }

func TestFetchAndPublish(t *testing.T) {
	settings := Settings{
		SeriesSlugs:  []string{"wec"},
		Categories:   map[string]struct{}{lightsouts.CategoryRace: {}, lightsouts.CategoryQualifying: {}},
		UpcomingDays: 90,
	}
	sink := &mockSink{topics: map[string]any{}}
	_, err := FetchAndPublish(context.Background(), settings, &mockFetcher{sessions: sampleSessions()}, sink)
	if err != nil {
		t.Fatal(err)
	}
	if sink.topics["summary"] == nil {
		t.Fatal("missing summary")
	}
}
