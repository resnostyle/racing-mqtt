package racing

import "github.com/resnostyle/mqttkit/mqttpub"

const virDeviceUID = "racing_mqtt_vir"

func BuildVirDiscoveryConfigs(topicPrefix string) []mqttpub.Config {
	return buildLiveNextDiscovery(topicPrefix, liveNextMeta{
		DeviceUID:  virDeviceUID,
		DeviceName: "VIR MQTT",
		Model:      "Virginia International Raceway",
		LiveName:   "VIR live event",
		NextName:   "Next VIR event",
		LiveIcon:   "mdi:flag-checkered",
	})
}
