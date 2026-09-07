package drift

import (
	"testing"
	"time"
)

func TestActiveAndNextEvent(t *testing.T) {
	start := time.Date(2026, 9, 24, 14, 0, 0, 0, time.UTC)
	events := []Event{{
		UID:   "fd:1",
		Name:  "HIGH STAKES",
		Start: start,
		End:   start.Add(72 * time.Hour),
	}}
	now := start.Add(2 * time.Hour)
	active := ActiveEvent(events, now)
	if active == nil || active.Name != "HIGH STAKES" {
		t.Fatalf("active %+v", active)
	}
	now = start.Add(-24 * time.Hour)
	next := NextEvent(events, now)
	if next == nil || next.Name != "HIGH STAKES" {
		t.Fatalf("next %+v", next)
	}
}

func TestParseDateRange(t *testing.T) {
	html := `<span>SEP 24 - SEP 26</span>`
	start, end, ok := parseDateRange(html, 2026)
	if !ok {
		t.Fatal("expected ok")
	}
	if start.Day() != 24 {
		t.Fatalf("start %v", start)
	}
	if !end.After(time.Date(2026, 9, 26, 23, 0, 0, 0, time.UTC)) {
		t.Fatalf("end %v", end)
	}
}
