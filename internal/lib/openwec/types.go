package openwec

import "time"

type Series struct {
	ID   int    `json:"id"`
	Key  string `json:"key"`
	Name string `json:"name"`
}

type Season struct {
	ID    int    `json:"id"`
	RawID string `json:"raw_id"`
	Year  int    `json:"year"`
	Label string `json:"label"`
}

type Event struct {
	ID    int    `json:"id"`
	RawID string `json:"raw_id"`
	Name  string `json:"name"`
	Round int    `json:"round"`
}

type Session struct {
	ID           int     `json:"id"`
	RawID        string  `json:"raw_id"`
	Name         string  `json:"name"`
	SessionType  string  `json:"session_type"`
	SessionAt    string  `json:"session_at"`
	IMSASeries   *string `json:"imsa_series"`
	SnapshotHour *int    `json:"snapshot_hour"`
}

type Driver struct {
	Slot      int     `json:"slot"`
	FirstName string  `json:"first_name"`
	LastName  string  `json:"last_name"`
	Country   *string `json:"country"`
}

type Result struct {
	Position     int      `json:"position"`
	CarNumber    string   `json:"car_number"`
	CarClass     string   `json:"car_class"`
	Vehicle      string   `json:"vehicle"`
	Team         string   `json:"team"`
	TyreSupplier *string  `json:"tyre_supplier"`
	Status       string   `json:"status"`
	LapsComplete int      `json:"laps_completed"`
	TotalTimeS   *float64 `json:"total_time_s"`
	GapToFirstS  *float64 `json:"gap_to_first_s"`
	FLLapNumber  *int     `json:"fl_lap_number"`
	FLTimeS      *float64 `json:"fl_time_s"`
	FLKPH        *float64 `json:"fl_kph"`
	Drivers      []Driver `json:"drivers"`
}

// ParsedSessionAt parses OpenWEC session_at timestamps.
func (s Session) ParsedSessionAt() (time.Time, error) {
	// API uses "2006-01-02 15:04:05+00" or similar.
	layouts := []string{
		"2006-01-02 15:04:05-07",
		"2006-01-02 15:04:05-07:00",
		time.RFC3339,
	}
	for _, layout := range layouts {
		if t, err := time.Parse(layout, s.SessionAt); err == nil {
			return t.UTC(), nil
		}
	}
	return time.Time{}, &parseError{value: s.SessionAt}
}

type parseError struct{ value string }

func (e *parseError) Error() string { return "openwec: cannot parse time " + e.value }
