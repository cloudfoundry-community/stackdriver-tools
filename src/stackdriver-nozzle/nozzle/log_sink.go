package nozzle

import (
	"encoding/json"
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
		Payload: lh.buildPayload(envelope),
		Labels:  lh.labelMaker.Build(envelope),
	}

	lh.logAdapter.PostLog(log)
	return nil
}

func structToMap(obj interface{}) map[string]interface{} {
	payload_json, _ := json.Marshal(obj)
	var unmarshaled_map map[string]interface{}
	json.Unmarshal(payload_json, &unmarshaled_map)

	return unmarshaled_map
}

func (ls *logSink) buildPayload(envelope *events.Envelope) interface{} {
	envelopeMap := structToMap(envelope)
	envelopeMap["eventType"] = envelope.GetEventType().String()

	// The json marshaling causes a loss in precision
	if envelope.GetTimestamp() != 0 {
		envelopeMap["timestamp"] = envelope.GetTimestamp()
	}

	switch envelope.GetEventType() {
	case events.Envelope_LogMessage:
		logMessage := envelope.GetLogMessage()
		logMessageMap := structToMap(logMessage)
		if logMessageMap != nil {
			// This is snake_cased to match the field in the protobuf. The other
			// fields we pass to Stackdriver are camelCased. We arbitrarily chose
			// to remain consistent with the protobuf.
			logMessageMap["message_type"] = logMessage.GetMessageType().String()
			logMessageMap["message"] = string(logMessage.GetMessage())
			envelopeMap["logMessage"] = logMessageMap
		}
	case events.Envelope_HttpStartStop:
		httpStartStop := envelope.GetHttpStartStop()
		httpStartStopMap := structToMap(httpStartStop)
		if httpStartStopMap != nil {
			httpStartStopMap["method"] = httpStartStop.GetMethod().String()
			httpStartStopMap["peerType"] = httpStartStop.GetPeerType().String()
			envelopeMap["httpStartStop"] = httpStartStopMap
		}
	}

	return envelopeMap
}
