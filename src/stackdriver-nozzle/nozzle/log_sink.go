package nozzle

import (
	"encoding/json"

	"cloud.google.com/go/logging"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/stackdriver"
	"github.com/cloudfoundry/sonde-go/events"
	"strings"
)

func NewLogSink(labelMaker LabelMaker, logAdapter stackdriver.LogAdapter, newlineToken string) Sink {
	return &logSink{
		labelMaker:   labelMaker,
		logAdapter:   logAdapter,
		newlineToken: newlineToken,
	}
}

type logSink struct {
	labelMaker   LabelMaker
	logAdapter   stackdriver.LogAdapter
	newlineToken string
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
			message := ls.parseMessage(logMessage.GetMessage())

			// This is snake_cased to match the field in the protobuf. The other
			// fields we pass to Stackdriver are camelCased. We arbitrarily chose
			// to remain consistent with the protobuf.
			logMessageMap["message_type"] = logMessage.GetMessageType().String()
			severity = parseSeverity(logMessage.GetMessageType())
			logMessageMap["message"] = message
			envelopeMap["logMessage"] = logMessageMap

			// Duplicate the message payload where stackdriver expects it
			envelopeMap["message"] = message
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

func (ls *logSink) parseMessage(rawMessage []byte) string {
	message := string(rawMessage)
	if ls.newlineToken != "" {
		message = strings.Replace(message, ls.newlineToken, "\n", -1)
	}
	return message
}

func parseSeverity(messageType events.LogMessage_MessageType) logging.Severity {
	if messageType == events.LogMessage_ERR {
		return logging.Error
	}

	return logging.Default
}
