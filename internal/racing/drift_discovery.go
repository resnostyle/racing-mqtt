package racing

import "github.com/resnostyle/mqttkit/mqttpub"

const driftDeviceUID = "racing_mqtt_drift"

func BuildDriftDiscoveryConfigs(topicPrefix string) []mqttpub.Config {
	return buildLiveNextDiscovery(topicPrefix, liveNextMeta{
		DeviceUID:  driftDeviceUID,
		DeviceName: "Formula Drift MQTT",
		Model:      "Drifting",
		LiveName:   "FD live event",
		NextName:   "Next Formula Drift round",
		LiveIcon:   "mdi:smoke",
	})
}
