package racing

import "github.com/resnostyle/mqttkit/mqttpub"

// liveNextMeta parameterizes the shared Drift/VIR live + next discovery sensors.
type liveNextMeta struct {
	DeviceUID  string
	DeviceName string
	Model      string
	LiveName   string
	NextName   string
	LiveIcon   string
}

func buildLiveNextDiscovery(topicPrefix string, meta liveNextMeta) []mqttpub.Config {
	summary := topicPrefix + "/summary"
	device := deviceBlockFor(meta.DeviceUID, meta.DeviceName, meta.Model)

	live := map[string]any{
		"name":                  meta.LiveName,
		"unique_id":             meta.DeviceUID + "_live",
		"state_topic":           summary,
		"value_template":        "{{ value_json.live }}",
		"payload_on":            "true",
		"payload_off":           "false",
		"device":                device,
		"object_id":             meta.DeviceUID + "_live",
		"icon":                  meta.LiveIcon,
		"json_attributes_topic": summary,
	}
	next := map[string]any{
		"name":                  meta.NextName,
		"unique_id":             meta.DeviceUID + "_next",
		"state_topic":           summary,
		"value_template":        "{{ value_json.next.start }}",
		"device":                device,
		"object_id":             meta.DeviceUID + "_next",
		"device_class":          "timestamp",
		"icon":                  "mdi:timer-outline",
		"json_attributes_topic": summary,
	}

	return []mqttpub.Config{
		{ObjectID: meta.DeviceUID + "_live", Component: "binary_sensor", Payload: live},
		{ObjectID: meta.DeviceUID + "_next", Component: "sensor", Payload: next},
	}
}
