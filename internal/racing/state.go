package racing

import (
	"sort"
	"time"

	"github.com/resnostyle/racing-mqtt/internal/lib/lightsouts"
)

const nearBoundaryWindow = 5 * time.Minute
const nearBoundaryPoll = 30 * time.Second

// ActiveSession returns the session live at now, if any.
func ActiveSession(sessions []lightsouts.Session, now time.Time) *lightsouts.Session {
	for i := range sessions {
		s := &sessions[i]
		if !s.Start.After(now) && now.Before(s.End) {
			return s
		}
	}
	return nil
}

// NextSession returns the earliest session starting after now.
func NextSession(sessions []lightsouts.Session, now time.Time) *lightsouts.Session {
	var next *lightsouts.Session
	for i := range sessions {
		s := &sessions[i]
		if !s.Start.After(now) {
			continue
		}
		if next == nil || s.Start.Before(next.Start) {
			next = s
		}
	}
	return next
}

// NextBoundary returns the nearest future session start or end.
func NextBoundary(sessions []lightsouts.Session, now time.Time) *time.Time {
	var nearest *time.Time
	for i := range sessions {
		s := &sessions[i]
		for _, t := range []time.Time{s.Start, s.End} {
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

// UpcomingSessions returns sessions starting within horizon from now.
func UpcomingSessions(sessions []lightsouts.Session, now time.Time, horizon time.Duration) []lightsouts.Session {
	limit := now.Add(horizon)
	out := make([]lightsouts.Session, 0)
	for _, s := range sessions {
		if s.Start.After(now) && !s.Start.After(limit) {
			out = append(out, s)
		}
	}
	sort.Slice(out, func(i, j int) bool {
		return out[i].Start.Before(out[j].Start)
	})
	return out
}

// NextBySeries returns the earliest upcoming session per series short name.
func NextBySeries(sessions []lightsouts.Session, now time.Time) map[string]lightsouts.Session {
	out := make(map[string]lightsouts.Session)
	for i := range sessions {
		s := sessions[i]
		if !s.Start.After(now) {
			continue
		}
		key := s.SeriesShort
		if key == "" {
			key = s.SeriesName
		}
		if existing, ok := out[key]; ok && !s.Start.Before(existing.Start) {
			continue
		}
		out[key] = s
	}
	return out
}

// PollInterval returns how long to wait before the next fetch.
func PollInterval(settings Settings, sessions []lightsouts.Session, now time.Time) time.Duration {
	return boundaryAwarePoll(settings.PollIntervalSeconds, NextBoundary(sessions, now), now)
}
