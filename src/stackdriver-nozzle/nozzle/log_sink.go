package nozzle

import (
	"encoding/json"

	"cloud.google.com/go/logging"
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
	payload, severity := lh.parseEnvelope(envelope)
	log := &stackdriver.Log{
		Payload:  payload,
		Labels:   lh.labelMaker.Build(envelope),
		Severity: severity,
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

func (ls *logSink) parseEnvelope(envelope *events.Envelope) (interface{}, logging.Severity) {
	envelopeMap := structToMap(envelope)
	envelopeMap["eventType"] = envelope.GetEventType().String()

	severity := logging.Default

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
			severity = parseSeverity(logMessage.GetMessageType())
			logMessageMap["message"] = string(logMessage.GetMessage())
			envelopeMap["logMessage"] = logMessageMap
			// Duplicate the message payload where stackdriver expects it
			envelopeMap["message"] = string(logMessage.GetMessage())
		}
	case events.Envelope_Error:
		errorMessage := envelope.GetError().GetMessage()
		envelopeMap["message"] = errorMessage
		severity = logging.Error
	case events.Envelope_HttpStartStop:
		httpStartStop := envelope.GetHttpStartStop()
		httpStartStopMap := structToMap(httpStartStop)
		if httpStartStopMap != nil {
			httpStartStopMap["method"] = httpStartStop.GetMethod().String()
			httpStartStopMap["peerType"] = httpStartStop.GetPeerType().String()
			envelopeMap["httpStartStop"] = httpStartStopMap
		}
	}

	return envelopeMap, severity
}

func parseSeverity(messageType events.LogMessage_MessageType) logging.Severity {
	if messageType == events.LogMessage_ERR {
		return logging.Error
	}

	return logging.Default
}
