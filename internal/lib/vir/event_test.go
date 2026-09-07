package vir

import (
	"testing"
	"time"
)

func TestUpcomingEvents(t *testing.T) {
	now := time.Date(2026, 8, 1, 0, 0, 0, 0, time.UTC)
	events := []Event{
		{Start: time.Date(2026, 8, 21, 12, 0, 0, 0, time.UTC), End: now.Add(72 * time.Hour)},
		{Start: time.Date(2027, 8, 20, 12, 0, 0, 0, time.UTC), End: now.Add(96 * time.Hour)},
	}
	upcoming := UpcomingEvents(events, now, 120*24*time.Hour)
	if len(upcoming) != 1 || upcoming[0].Start.Year() != 2026 {
		t.Fatalf("upcoming %+v", upcoming)
	}
}
