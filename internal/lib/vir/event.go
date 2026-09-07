package vir

import "time"

const (
	Venue    = "Virginia International Raceway"
	Location = "Alton, Virginia, USA"
)

// Event is a scheduled event at Virginia International Raceway.
type Event struct {
	UID      string
	Series   string
	Name     string
	Slug     string
	Location string
	Venue    string
	Link     string
	Start    time.Time
	End      time.Time
	Source   string
}

// ActiveEvent returns the event live at now, if any.
func ActiveEvent(events []Event, now time.Time) *Event {
	for i := range events {
		e := &events[i]
		if !e.Start.After(now) && now.Before(e.End) {
			return e
		}
	}
	return nil
}

// NextEvent returns the earliest event starting after now.
func NextEvent(events []Event, now time.Time) *Event {
	var next *Event
	for i := range events {
		e := &events[i]
		if !e.Start.After(now) {
			continue
		}
		if next == nil || e.Start.Before(next.Start) {
			next = e
		}
	}
	return next
}

// NextBoundary returns the nearest future start or end across events.
func NextBoundary(events []Event, now time.Time) *time.Time {
	var nearest *time.Time
	for i := range events {
		e := &events[i]
		for _, t := range []time.Time{e.Start, e.End} {
			if !t.After(now) {
				continue
			}
			if nearest == nil || t.Before(*nearest) {
				copy := t
				nearest = &copy
			}
		}
	}
	return nearest
}

// UpcomingEvents returns events starting within horizon from now.
func UpcomingEvents(events []Event, now time.Time, horizon time.Duration) []Event {
	limit := now.Add(horizon)
	out := make([]Event, 0)
	for _, e := range events {
		if e.Start.After(now) && !e.Start.After(limit) {
			out = append(out, e)
		}
	}
	return out
}
