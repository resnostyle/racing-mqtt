package racing

import (
	"fmt"
	"time"

	"github.com/resnostyle/mqttkit/mqttpub"
)

// publishScheduleTopics publishes active/current, upcoming/current, and summary under topicRoot.
func publishScheduleTopics(mqtt mqttpub.Sink, topicRoot string, active, upcoming, summary map[string]any) error {
	if err := mqtt.Publish(topicRoot+"/active/current", active, true); err != nil {
		return fmt.Errorf("publish %s active: %w", topicRoot, err)
	}
	if err := mqtt.Publish(topicRoot+"/upcoming/current", upcoming, true); err != nil {
		return fmt.Errorf("publish %s upcoming: %w", topicRoot, err)
	}
	if err := mqtt.Publish(topicRoot+"/summary", summary, true); err != nil {
		return fmt.Errorf("publish %s summary: %w", topicRoot, err)
	}
	return nil
}

func boundaryAwarePoll(defaultSeconds int, boundary *time.Time, now time.Time) time.Duration {
	defaultInterval := time.Duration(defaultSeconds) * time.Second
	if boundary != nil && boundary.Sub(now) <= nearBoundaryWindow {
		return nearBoundaryPoll
	}
	return defaultInterval
}
