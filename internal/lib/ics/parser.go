package ics

import (
	"bufio"
	"strings"
	"time"
)

// Event is a parsed VEVENT component.
type Event struct {
	UID         string
	Summary     string
	Description string
	Location    string
	Start       time.Time
	End         time.Time
	AllDay      bool
}

// Parse decodes iCalendar text into events.
func Parse(data string) ([]Event, error) {
	data = unfoldLines(data)
	var events []Event
	var cur *Event
	lines := strings.Split(data, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "BEGIN:VEVENT" {
			cur = &Event{}
			continue
		}
		if line == "END:VEVENT" {
			if cur != nil && !cur.Start.IsZero() {
				if cur.End.IsZero() {
					cur.End = cur.Start.Add(24 * time.Hour)
				}
				events = append(events, *cur)
			}
			cur = nil
			continue
		}
		if cur == nil {
			continue
		}
		key, value, ok := strings.Cut(line, ":")
		if !ok {
			continue
		}
		prop, params, _ := strings.Cut(key, ";")
		prop = strings.ToUpper(prop)
		switch prop {
		case "UID":
			cur.UID = unescape(value)
		case "SUMMARY":
			cur.Summary = unescape(value)
		case "DESCRIPTION":
			cur.Description = unescape(value)
		case "LOCATION":
			cur.Location = unescape(value)
		case "DTSTART":
			t, allDay := parseICSTime(value, params)
			cur.Start = t
			cur.AllDay = allDay
		case "DTEND":
			t, allDay := parseICSTime(value, params)
			cur.End = t
			if allDay {
				cur.AllDay = true
			}
		}
	}
	return events, nil
}

func unfoldLines(data string) string {
	lines := strings.Split(strings.ReplaceAll(data, "\r\n", "\n"), "\n")
	var out []string
	for _, line := range lines {
		if strings.HasPrefix(line, " ") || strings.HasPrefix(line, "\t") {
			if len(out) > 0 {
				out[len(out)-1] += strings.TrimLeft(line, " \t")
			}
			continue
		}
		out = append(out, line)
	}
	return strings.Join(out, "\n")
}

func unescape(s string) string {
	s = strings.ReplaceAll(s, "\\n", "\n")
	s = strings.ReplaceAll(s, "\\,", ",")
	s = strings.ReplaceAll(s, "\\;", ";")
	return s
}

func parseICSTime(value, params string) (time.Time, bool) {
	params = strings.ToUpper(params)
	allDay := strings.Contains(params, "VALUE=DATE") || (!strings.Contains(value, "T") && len(value) == 8)
	if allDay {
		if len(value) >= 8 {
			t, err := time.Parse("20060102", value[:8])
			if err == nil {
				return t.UTC(), true
			}
		}
		return time.Time{}, true
	}
	layouts := []string{
		"20060102T150405Z",
		"20060102T150405",
		time.RFC3339,
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, value); err == nil {
			return t.UTC(), false
		}
	}
	return time.Time{}, false
}

// FilterSummary returns events whose summary or description matches any needle (case-insensitive).
func FilterSummary(events []Event, needles ...string) []Event {
	if len(needles) == 0 {
		return events
	}
	var out []Event
	for _, e := range events {
		hay := strings.ToLower(e.Summary + " " + e.Description + " " + e.Location)
		for _, n := range needles {
			if strings.Contains(hay, strings.ToLower(n)) {
				out = append(out, e)
				break
			}
		}
	}
	return out
}

// Scan is a convenience wrapper around Parse for streaming readers.
func Scan(r *bufio.Reader) ([]Event, error) {
	var b strings.Builder
	for {
		line, err := r.ReadString('\n')
		b.WriteString(line)
		if err != nil {
			break
		}
	}
	return Parse(b.String())
}
