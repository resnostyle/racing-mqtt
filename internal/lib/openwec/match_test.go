package openwec

import (
	"testing"
	"time"

	"github.com/resnostyle/racing-mqtt/internal/lib/lightsouts"
)

func TestNameSimilarity(t *testing.T) {
	if nameSimilarity("Circuit of the Americas", "CIRCUIT OF THE AMERICAS") < 0.9 {
		t.Fatal("expected high similarity for COTA")
	}
	if nameSimilarity("Fuji Speedway", "FUJI SPEEDWAY") < 0.9 {
		t.Fatal("expected high similarity for Fuji")
	}
}

func TestMatchEventBahrain(t *testing.T) {
	lo := lightsouts.Session{
		Category:  lightsouts.CategoryRace,
		Circuit:   "Bahrain International Circuit",
		EventName: "8 Hours of Bahrain",
		EventSlug: "8-hours-of-bahrain",
		Start:     time.Date(2025, 11, 8, 14, 0, 0, 0, time.UTC),
	}
	events := []Event{
		{Name: "LOSAIL", ID: 1},
		{Name: "BAHRAIN INTERNATIONAL CIRCUIT", ID: 109},
	}
	ev, score := MatchEvent(lo, events)
	if ev == nil || ev.Name != "BAHRAIN INTERNATIONAL CIRCUIT" {
		t.Fatalf("event %+v score %v", ev, score)
	}
	if score < MinEventMatchScore() {
		t.Fatalf("score %v", score)
	}
}

func TestMatchSessionRace(t *testing.T) {
	start := time.Date(2025, 11, 8, 14, 0, 0, 0, time.UTC)
	lo := lightsouts.Session{
		Category:    lightsouts.CategoryRace,
		SessionName: "Race",
		Circuit:     "Bahrain International Circuit",
		EventName:   "8 Hours of Bahrain",
		Start:       start,
	}
	events := []Event{{Name: "BAHRAIN INTERNATIONAL CIRCUIT", ID: 109}}
	sessions := []Session{
		{ID: 828, Name: "Qualifying", SessionType: "Qualifying", SessionAt: "2025-11-07 18:40:00+00"},
		{ID: 829, Name: "Race", SessionType: "Race", SessionAt: "2025-11-08 14:00:00+00"},
	}
	sess, ev, score := Match(lo, events, sessions)
	if sess == nil || sess.ID != 829 {
		t.Fatalf("session %+v", sess)
	}
	if ev == nil || ev.ID != 109 {
		t.Fatalf("event %+v", ev)
	}
	if score < 0.5 {
		t.Fatalf("score %v", score)
	}
}

func TestSeriesKeyForLightsouts(t *testing.T) {
	key, ok := SeriesKeyForLightsouts("wec")
	if !ok || key != "WEC" {
		t.Fatalf("got %q %v", key, ok)
	}
	key, ok = SeriesKeyForLightsouts("imsa-sportscar-championship")
	if !ok || key != "IMSA" {
		t.Fatalf("got %q %v", key, ok)
	}
}
