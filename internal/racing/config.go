package racing

import (
	"fmt"
	"os"
	"strings"

	"github.com/resnostyle/mqttkit/env"
	"github.com/resnostyle/racing-mqtt/internal/lib/lightsouts"
	"github.com/resnostyle/racing-mqtt/internal/lib/openwec"
)

type Settings struct {
	env.MQTT
	SeriesSlugs                []string
	Categories                 map[string]struct{}
	PollIntervalSeconds        int
	UpcomingDays               int
	OpenWECEnabled             bool
	OpenWECBaseURL             string
	OpenWECAPIKey              string
	OpenWECResultsLookbackDays int
	DriftEnabled               bool
	DriftSource                string
	DriftICSURL                string
	DriftFormuladURL           string
	VirEnabled                 bool
	VirEventsURL               string
	VirWPPostsURL              string
	VirWPCategoryID            int
}

func FromEnv() (Settings, error) {
	mqtt, err := env.LoadMQTT("home/racing", "racing-mqtt")
	if err != nil {
		return Settings{}, err
	}
	slugs, err := parseSlugs(os.Getenv("RACING_SERIES_SLUGS"))
	if err != nil {
		return Settings{}, err
	}
	categories := parseCategories(os.Getenv("RACING_CATEGORIES"))
	poll, err := env.Int("RACING_POLL_INTERVAL_SECONDS", 900)
	if err != nil {
		return Settings{}, err
	}
	if poll < 30 {
		return Settings{}, fmt.Errorf("RACING_POLL_INTERVAL_SECONDS must be >= 30")
	}
	days, err := env.Int("RACING_UPCOMING_DAYS", 90)
	if err != nil {
		return Settings{}, err
	}
	if days < 1 {
		return Settings{}, fmt.Errorf("RACING_UPCOMING_DAYS must be >= 1")
	}
	lookback, err := env.Int("OPENWEC_RESULTS_LOOKBACK_DAYS", 30)
	if err != nil {
		return Settings{}, err
	}
	if lookback < 1 {
		return Settings{}, fmt.Errorf("OPENWEC_RESULTS_LOOKBACK_DAYS must be >= 1")
	}
	driftSource := strings.ToLower(strings.TrimSpace(env.Get("DRIFT_SOURCE", "formulad")))
	if driftSource != "formulad" && driftSource != "ics" {
		return Settings{}, fmt.Errorf("DRIFT_SOURCE must be formulad or ics")
	}
	virCategory, err := env.Int("VIR_WP_CATEGORY_ID", 81)
	if err != nil {
		return Settings{}, err
	}
	if virCategory < 1 {
		return Settings{}, fmt.Errorf("VIR_WP_CATEGORY_ID must be >= 1")
	}
	return Settings{
		MQTT:                       mqtt,
		SeriesSlugs:                slugs,
		Categories:                 categories,
		PollIntervalSeconds:        poll,
		UpcomingDays:               days,
		OpenWECEnabled:             env.Bool("OPENWEC_ENABLED", true),
		OpenWECBaseURL:             env.Get("OPENWEC_BASE_URL", openwec.DefaultBaseURL),
		OpenWECAPIKey:              strings.TrimSpace(os.Getenv("OPENWEC_API_KEY")),
		OpenWECResultsLookbackDays: lookback,
		DriftEnabled:               env.Bool("DRIFT_ENABLED", true),
		DriftSource:                driftSource,
		DriftICSURL:                strings.TrimSpace(os.Getenv("DRIFT_ICS_URL")),
		DriftFormuladURL:           strings.TrimSpace(os.Getenv("DRIFT_FORMULAD_URL")),
		VirEnabled:                 env.Bool("VIR_ENABLED", true),
		VirEventsURL:               strings.TrimSpace(os.Getenv("VIR_EVENTS_URL")),
		VirWPPostsURL:              strings.TrimSpace(os.Getenv("VIR_WP_POSTS_URL")),
		VirWPCategoryID:            virCategory,
	}, nil
}

func parseSlugs(raw string) ([]string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return []string{
			"wec",
			"imsa-sportscar-championship",
			"wrc",
			"supercars-championship",
		}, nil
	}
	var slugs []string
	seen := map[string]struct{}{}
	for _, part := range strings.Split(raw, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		if _, dup := seen[part]; dup {
			continue
		}
		seen[part] = struct{}{}
		slugs = append(slugs, part)
	}
	if len(slugs) == 0 {
		return nil, fmt.Errorf("RACING_SERIES_SLUGS is empty")
	}
	return slugs, nil
}

func parseCategories(raw string) map[string]struct{} {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return map[string]struct{}{
			lightsouts.CategoryRace:       {},
			lightsouts.CategoryQualifying: {},
		}
	}
	out := make(map[string]struct{})
	for _, part := range strings.Split(raw, ",") {
		part = strings.ToLower(strings.TrimSpace(part))
		if part != "" {
			out[part] = struct{}{}
		}
	}
	return out
}

func (s Settings) FilterSessions(sessions []lightsouts.Session) []lightsouts.Session {
	if len(s.Categories) == 0 {
		return sessions
	}
	out := make([]lightsouts.Session, 0, len(sessions))
	for _, sess := range sessions {
		if _, ok := s.Categories[sess.Category]; ok {
			out = append(out, sess)
		}
	}
	return out
}
