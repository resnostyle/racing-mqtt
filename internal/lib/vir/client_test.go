package vir

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

const eventsFixture = `<!DOCTYPE html><html><body>
<h2 class="ha-card-title">IMSA Michelin GT Challenge and VA is for Racing Lovers Grand Prix: Aug 21-23, 2026</h2>
<h2 class="ha-card-title">Racing America: Sept 17-20, 2026</h2>
<h2 class="ha-card-title">Veterans Race of Remembrance: Nov 6-8, 2026</h2>
<h2 class="ha-card-title">Charity Laps for Victory Junction: Nov 20, 2026</h2>
<h2 class="ha-card-title">IMSA Michelin GT Challenge and VA is for Racing Lovers Grand Prix: Aug 20-22, 2027</h2>
</body></html>`

func TestParseEventsPage(t *testing.T) {
	events, err := parseEventsPage(eventsFixture)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 5 {
		t.Fatalf("got %d events", len(events))
	}
	if events[0].Series != "IMSA" || events[0].Slug == "" {
		t.Fatalf("first event %+v", events[0])
	}
	if events[0].Venue != Venue || events[0].Location != Location {
		t.Fatalf("location %+v", events[0])
	}
	loc, _ := time.LoadLocation(virTZ)
	wantStart := time.Date(2026, 8, 21, 8, 0, 0, 0, loc)
	if !events[0].Start.Equal(wantStart.UTC()) {
		t.Fatalf("start got %v want %v", events[0].Start, wantStart.UTC())
	}
	if events[3].Start.In(loc).Day() != 20 {
		t.Fatalf("single-day start %+v", events[3])
	}
}

func TestActiveAndNextEvent(t *testing.T) {
	events, err := parseEventsPage(eventsFixture)
	if err != nil {
		t.Fatal(err)
	}
	now := events[0].Start.Add(2 * time.Hour)
	if active := ActiveEvent(events, now); active == nil || active.Series != "IMSA" {
		t.Fatalf("active %+v", active)
	}
	later := events[1].Start.Add(-time.Hour)
	if next := NextEvent(events, later); next == nil || next.Series != "Racing America" {
		t.Fatalf("next %+v", next)
	}
}

func TestFetchEventsWithMockServer(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case stringsHasSuffix(r.URL.Path, "/events/"):
			_, _ = w.Write([]byte(eventsFixture))
		case stringsHasSuffix(r.URL.Path, "/posts"):
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`[{"link":"https://virnow.com/imsa/","excerpt":{"rendered":"<p>IMSA Michelin GT Challenge August 21-23, 2026 Learn More</p>"}}]`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	client := NewClient(srv.URL+"/events/", srv.URL+"/wp-json/wp/v2/posts", 81)
	events, err := client.FetchEvents(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 5 {
		t.Fatalf("got %d events", len(events))
	}
	if events[0].Link != "https://virnow.com/imsa/" {
		t.Fatalf("link %q", events[0].Link)
	}
}

func stringsHasSuffix(s, suffix string) bool {
	return len(s) >= len(suffix) && s[len(s)-len(suffix):] == suffix
}
