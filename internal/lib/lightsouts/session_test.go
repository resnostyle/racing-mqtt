package lightsouts

import (
	"testing"
	"time"
)

func TestClassifySession(t *testing.T) {
	cases := map[string]string{
		"Free Practice 1": CategoryPractice,
		"Qualifying":      CategoryQualifying,
		"Sprint":          CategorySprint,
		"Race":            CategoryRace,
		"Hyperpole":       CategoryQualifying,
		"Warm Up":         CategoryPractice,
		"Other Event":     CategoryOther,
	}
	for name, want := range cases {
		if got := ClassifySession(name); got != want {
			t.Fatalf("%q: got %q want %q", name, got, want)
		}
	}
}

func TestParseSessionStart(t *testing.T) {
	start, ok := parseSessionStart("2026-09-06", "18:00")
	if !ok {
		t.Fatal("expected ok")
	}
	if start.Format(time.RFC3339) != "2026-09-06T18:00:00Z" {
		t.Fatalf("got %s", start.Format(time.RFC3339))
	}

	start, ok = parseSessionStart("2026-08-27T22:00:00Z", "")
	if !ok {
		t.Fatal("expected ok for RFC3339 date")
	}
	if start.UTC().Hour() != 22 {
		t.Fatalf("got hour %d", start.UTC().Hour())
	}
}

func TestFlattenSeries(t *testing.T) {
	payload := map[string]any{
		"name":    "WEC",
		"nameAlt": nil,
		"slug":    "wec",
		"events": []any{
			map[string]any{
				"name": "Lone Star Le Mans",
				"slug": "lone-star-le-mans",
				"tba":  false,
				"circuit": map[string]any{
					"name": "Circuit of the Americas",
					"country": map[string]any{
						"name": "United States",
					},
				},
				"sessions": []any{
					map[string]any{
						"id":       float64(1607298),
						"name":     "Race",
						"date":     "2026-09-06",
						"time":     "18:00",
						"duration": float64(360),
						"main":     true,
					},
				},
			},
		},
	}
	sessions := FlattenSeries(payload)
	if len(sessions) != 1 {
		t.Fatalf("got %d sessions", len(sessions))
	}
	s := sessions[0]
	if s.SeriesShort != "WEC" {
		t.Fatalf("series %q", s.SeriesShort)
	}
	if s.Category != CategoryRace {
		t.Fatalf("category %q", s.Category)
	}
	if !s.IsMain {
		t.Fatal("expected main session")
	}
	if s.Circuit != "Circuit of the Americas" {
		t.Fatalf("circuit %q", s.Circuit)
	}
}
