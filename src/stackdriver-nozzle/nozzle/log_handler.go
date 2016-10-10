package nozzle

import (
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/stackdriver"
	"github.com/cloudfoundry/sonde-go/events"
)

type LogHandler interface {
	HandleLog(*events.Envelope)
}

type LabelMakerFn func(*events.Envelope) map[string]string

func NewLogHandler(labelMaker LabelMakerFn, logAdapter stackdriver.LogAdapter) LogHandler {
	return &logHandler{labelMaker: labelMaker, logAdapter: logAdapter}
}

type logHandler struct {
	labelMaker LabelMakerFn
	logAdapter stackdriver.LogAdapter
}

func (lh *logHandler) HandleLog(envelope *events.Envelope) {
	log := &stackdriver.Log{
		Payload: envelope,
		Labels:  lh.labelMaker(envelope),
	}

	lh.logAdapter.PostLog(log)
}
