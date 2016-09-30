package nozzle

import (
	"fmt"

	"github.com/evandbrown/gcp-tools-release/src/stackdriver-nozzle/stackdriver"
)

type Nozzle struct {
	StackdriverClient stackdriver.Client
}

func (n *Nozzle) Connect() bool {
	return true
}

func (n *Nozzle) ShipEvents(event map[string]interface{}, _ string /* TODO research second string */) {
	switch event["event_type"] {

	case "ValueMetric":
		name := event["name"]
		count := event["value"]
		n.StackdriverClient.PostMetric(name.(string), count.(float64))

	default:
		n.StackdriverClient.PostLog(event, map[string]string{
			"event_type": fmt.Sprintf("%v", event["event_type"]),
		})
	}

}
