package racing

import (
	"testing"

	"github.com/resnostyle/racing-mqtt/internal/lib/openwec"
)

func TestDedupeResults(t *testing.T) {
	raw := []openwec.Result{
		{CarClass: "HYPERCAR", CarNumber: "7", Position: 1, Team: "Toyota"},
		{CarClass: "HYPERCAR", CarNumber: "7", Position: 1, Team: "Toyota"},
		{CarClass: "LMGT3", CarNumber: "27", Position: 1, Team: "Heart of Racing"},
	}
	out := dedupeResults(raw)
	if len(out) != 2 {
		t.Fatalf("got %d", len(out))
	}
}

func TestBuildResultsPayload(t *testing.T) {
	lo := sampleSessions()[1]
	payload := BuildResultsPayload(lo, "WEC", &openwec.Event{ID: 109, Name: "BAHRAIN INTERNATIONAL CIRCUIT"},
		&openwec.Session{ID: 829, SessionAt: "2025-11-08 14:00:00+00"},
		[]openwec.Result{
			{Position: 1, CarNumber: "7", CarClass: "HYPERCAR", Team: "Toyota Gazoo Racing", Drivers: []openwec.Driver{{FirstName: "Mike", LastName: "CONWAY"}}},
		},
	)
	if payload["series"] != "WEC" {
		t.Fatalf("series %v", payload["series"])
	}
	podium, ok := payload["podium_by_class"].(map[string]any)
	if !ok || len(podium) != 1 {
		t.Fatalf("podium %+v", payload["podium_by_class"])
	}
	msg := FormatWinnerMessage(payload)
	if msg == "" {
		t.Fatal("expected winner message")
	}
}
