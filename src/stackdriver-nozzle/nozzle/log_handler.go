package nozzle

import (
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/stackdriver"
	"github.com/cloudfoundry/sonde-go/events"
)

type LogHandler interface {
	HandleEnvelope(*events.Envelope)
}

func NewLogHandler(labelMaker LabelMaker, logAdapter stackdriver.LogAdapter) LogHandler {
	return &logHandler{labelMaker: labelMaker, logAdapter: logAdapter}
}

type logHandler struct {
	labelMaker LabelMaker
	logAdapter stackdriver.LogAdapter
}

func (lh *logHandler) HandleEnvelope(envelope *events.Envelope) {
	log := &stackdriver.Log{
		Payload: envelope,
		Labels:  lh.labelMaker.Build(envelope),
	}

	lh.logAdapter.PostLog(log)
}
