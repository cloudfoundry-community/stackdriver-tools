package filter

import (
	"github.com/cloudfoundry/sonde-go/events"
	"github.com/evandbrown/gcp-tools-release/src/stackdriver-nozzle/firehose"
)

type filter struct {
	destination firehose.FirehoseHandler
}

func New() firehose.FirehoseHandler {
	return &filter{}
}

func (f *filter) HandleEvent(*events.Envelope) error {
	return nil
}
