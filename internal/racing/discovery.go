package racing

import (
	"fmt"
	"strings"

	"github.com/resnostyle/mqttkit/hadisc"
	"github.com/resnostyle/mqttkit/mqttpub"
)

const (
	deviceManufacturer = "racing-mqtt"
	deviceUID          = "racing_mqtt"
)

type seriesDiscoveryMeta struct {
	NextKey string
	Label   string
}

// nextBySeriesKeys maps Lightsouts series slugs to summary.next_by_series keys.
var nextBySeriesKeys = map[string]seriesDiscoveryMeta{
	"wec":                         {NextKey: "WEC", Label: "WEC"},
	"imsa-sportscar-championship": {NextKey: "IMSA", Label: "IMSA"},
	"wrc":                         {NextKey: "WRC", Label: "WRC"},
	"supercars-championship":      {NextKey: "Supercars Championship", Label: "Supercars"},
}

func deviceBlock() map[string]any {
	return deviceBlockFor(deviceUID, "Racing MQTT", "Motorsport")
}

func deviceBlockFor(uid, name, model string) map[string]any {
	return hadisc.Device([]string{uid}, name, deviceManufacturer, model)
}

func BuildDiscoveryConfigs(topicPrefix string, seriesSlugs []string) []mqttpub.Config {
	summary := topicPrefix + "/summary"
	device := deviceBlock()

	live := map[string]any{
		"name":                  "Live session",
		"unique_id":             deviceUID + "_live",
		"state_topic":           summary,
		"value_template":        "{{ value_json.live }}",
		"payload_on":            "true",
		"payload_off":           "false",
		"device":                device,
		"object_id":             deviceUID + "_live",
		"icon":                  "mdi:flag-checkered",
		"json_attributes_topic": summary,
	}

	configs := []mqttpub.Config{
		{ObjectID: deviceUID + "_live", Component: "binary_sensor", Payload: live},
	}
	for _, slug := range seriesSlugs {
		meta, ok := nextBySeriesKeys[slug]
		if !ok {
			continue
		}
		objectID := deviceUID + "_next_" + strings.ReplaceAll(slug, "-", "_")
		next := map[string]any{
			"name":                  fmt.Sprintf("Next %s session", meta.Label),
			"unique_id":             objectID,
			"state_topic":           summary,
			"value_template":        nextSeriesValueTemplate(meta.NextKey),
			"device":                device,
			"object_id":             objectID,
			"device_class":          "timestamp",
			"icon":                  "mdi:timer-outline",
			"json_attributes_topic": summary,
		}
		configs = append(configs, mqttpub.Config{
			ObjectID:  objectID,
			Component: "sensor",
			Payload:   next,
		})
	}
	return configs
}

func nextSeriesValueTemplate(nextKey string) string {
	if strings.ContainsAny(nextKey, " -") {
		return fmt.Sprintf("{{ value_json.next_by_series['%s'].start }}", nextKey)
	}
	return fmt.Sprintf("{{ value_json.next_by_series.%s.start }}", nextKey)
}
