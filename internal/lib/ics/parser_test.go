package ics

import (
	"testing"
	"time"
)

const sampleICS = `BEGIN:VCALENDAR
VERSION:2.0
BEGIN:VEVENT
UID:fd-test-1
SUMMARY:Formula Drift 2026 Round 7 Las Vegas
DTSTART:20260924T140000Z
DTEND:20260927T020000Z
LOCATION:Las Vegas Motor Speedway
DESCRIPTION:Organised by Formula Drift
END:VEVENT
END:VCALENDAR
`

func TestParseICS(t *testing.T) {
	events, err := Parse(sampleICS)
	if err != nil {
		t.Fatal(err)
	}
	if len(events) != 1 {
		t.Fatalf("got %d events", len(events))
	}
	e := events[0]
	if e.Summary == "" || e.UID != "fd-test-1" {
		t.Fatalf("event %+v", e)
	}
	if e.Start.UTC().Format(time.RFC3339) != "2026-09-24T14:00:00Z" {
		t.Fatalf("start %s", e.Start)
	}
}

func TestFilterSummary(t *testing.T) {
	events, _ := Parse(`BEGIN:VCALENDAR
BEGIN:VEVENT
UID:1
SUMMARY:D1 Grand Prix Okayama
DTSTART:20260101T120000Z
DTEND:20260101T180000Z
END:VEVENT
BEGIN:VEVENT
UID:2
SUMMARY:Formula Drift Round 1
DTSTART:20260410T120000Z
DTEND:20260411T180000Z
END:VEVENT
END:VCALENDAR`)
	out := FilterSummary(events, "formula drift")
	if len(out) != 1 {
		t.Fatalf("got %d", len(out))
	}
}
