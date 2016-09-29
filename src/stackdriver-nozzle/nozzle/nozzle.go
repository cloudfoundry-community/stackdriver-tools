package nozzle

import (
	"github.com/evandbrown/gcp-tools-release/src/stackdriver-nozzle/stackdriver"
	"fmt"
)


type Nozzle struct {
	StackdriverClient stackdriver.Client
}

func (n *Nozzle) Connect() bool {
	return true
}

func (n *Nozzle) ShipEvents(event map[string]interface{}, _ string /* TODO research second string */) {
	n.StackdriverClient.Post(event, map[string]string{
		"event_type": fmt.Sprintf("%v", event["event_type"]),
	})
}
