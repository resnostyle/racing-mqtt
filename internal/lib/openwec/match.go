package openwec

import (
	"strings"
	"time"

	"github.com/resnostyle/racing-mqtt/internal/lib/lightsouts"
)

const (
	sessionTimeTolerance = 4 * time.Hour
	minEventMatchScore   = 0.45
)

// MinEventMatchScore returns the minimum event name similarity for a match.
func MinEventMatchScore() float64 { return minEventMatchScore }

// MatchEvent finds the best OpenWEC event for a Lightsouts session.
func MatchEvent(lo lightsouts.Session, events []Event) (*Event, float64) {
	return bestEventMatch(lo, events)
}

// Match finds the OpenWEC race session for a Lightsouts session.
func Match(lo lightsouts.Session, events []Event, sessions []Session) (*Session, *Event, float64) {
	if lo.Category != lightsouts.CategoryRace {
		return nil, nil, 0
	}
	bestEvent, eventScore := bestEventMatch(lo, events)
	if bestEvent == nil || eventScore < minEventMatchScore {
		return nil, nil, eventScore
	}
	bestSession, sessScore := bestSessionMatch(lo.Start, sessions)
	if bestSession == nil {
		return nil, bestEvent, eventScore
	}
	return bestSession, bestEvent, (eventScore + sessScore) / 2
}

func bestEventMatch(lo lightsouts.Session, events []Event) (*Event, float64) {
	var best *Event
	var bestScore float64
	for i := range events {
		score := eventMatchScore(lo, &events[i])
		if score > bestScore {
			bestScore = score
			best = &events[i]
		}
	}
	return best, bestScore
}

func eventMatchScore(lo lightsouts.Session, ev *Event) float64 {
	names := []string{lo.Circuit, lo.EventName, lo.EventSlug}
	eventName := ev.Name
	var best float64
	for _, name := range names {
		if name == "" {
			continue
		}
		if s := nameSimilarity(name, eventName); s > best {
			best = s
		}
	}
	// Known aliases where Lightsouts and OpenWEC names diverge.
	if best < minEventMatchScore {
		if alias, ok := circuitAliases[lo.EventSlug]; ok {
			if s := nameSimilarity(alias, eventName); s > best {
				best = s
			}
		}
	}
	return best
}

var circuitAliases = map[string]string{
	"lone-star-le-mans":              "circuit of the americas",
	"6-hours-of-fuji":                "fuji speedway",
	"6-hours-of-barcelona":           "barcelona",
	"6-hours-of-monza":               "monza",
	"petit-le-mans":                  "road atlanta",
	"sportscar-endurance-grand-prix": "road america",
	"gt-challenge-at-vir":            "virginia international",
	"battle-on-the-bricks":           "indianapolis",
	"24-hours-of-daytona":            "daytona",
}

func bestSessionMatch(start time.Time, sessions []Session) (*Session, float64) {
	var best *Session
	var bestDelta time.Duration
	for i := range sessions {
		s := &sessions[i]
		if !isRaceSession(s) {
			continue
		}
		at, err := s.ParsedSessionAt()
		if err != nil {
			continue
		}
		delta := at.Sub(start)
		if delta < 0 {
			delta = -delta
		}
		if delta > sessionTimeTolerance {
			continue
		}
		if best == nil || delta < bestDelta {
			best = s
			bestDelta = delta
		}
	}
	if best == nil {
		return nil, 0
	}
	score := 1.0 - float64(bestDelta)/float64(sessionTimeTolerance)
	return best, score
}

func isRaceSession(s *Session) bool {
	if strings.EqualFold(s.SessionType, "Race") {
		return true
	}
	return strings.EqualFold(strings.TrimSpace(s.Name), "Race")
}

func nameSimilarity(a, b string) float64 {
	na := normalizeName(a)
	nb := normalizeName(b)
	if na == "" || nb == "" {
		return 0
	}
	if na == nb {
		return 1
	}
	if strings.Contains(na, nb) || strings.Contains(nb, na) {
		return 0.9
	}
	ta := strings.Fields(na)
	tb := strings.Fields(nb)
	if len(ta) == 0 || len(tb) == 0 {
		return 0
	}
	shared := 0
	for _, wa := range ta {
		if len(wa) < 3 {
			continue
		}
		for _, wb := range tb {
			if wa == wb {
				shared++
				break
			}
		}
	}
	denom := len(ta)
	if len(tb) > denom {
		denom = len(tb)
	}
	return float64(shared) / float64(denom)
}

func normalizeName(s string) string {
	s = strings.ToLower(strings.TrimSpace(s))
	var b strings.Builder
	lastSpace := false
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			lastSpace = false
			continue
		}
		if !lastSpace {
			b.WriteByte(' ')
			lastSpace = true
		}
	}
	return strings.TrimSpace(b.String())
}
