package nozzle

import (
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/stackdriver"
	"github.com/cloudfoundry/sonde-go/events"
)

func NewLogSink(labelMaker LabelMaker, logAdapter stackdriver.LogAdapter) Sink {
	return &logSink{labelMaker: labelMaker, logAdapter: logAdapter}
}

type logSink struct {
	labelMaker LabelMaker
	logAdapter stackdriver.LogAdapter
}

func (lh *logSink) Receive(envelope *events.Envelope) error {
	log := &stackdriver.Log{
		Payload: envelope,
		Labels:  lh.labelMaker.Build(envelope),
	}

	lh.logAdapter.PostLog(log)
	return nil
}
