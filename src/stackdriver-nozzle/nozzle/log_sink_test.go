/*
 * Copyright 2017 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package nozzle_test

import (
	"time"

	"cloud.google.com/go/logging"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/mocks"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/nozzle"
	"github.com/cloudfoundry/sonde-go/events"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("LogSink", func() {
	var (
		subject    nozzle.Sink
		labelMaker *mocks.LabelMaker
		logAdapter *mocks.LogAdapter
		labels     map[string]string
	)

	BeforeEach(func() {
		labels = map[string]string{"foo": "bar", "applicationId": "ab313b25-aa48-4a8f-8e7d-d63a6d410e7c"}
		labelMaker = &mocks.LabelMaker{Labels: labels}
		logAdapter = &mocks.LogAdapter{}

		newlineToken := ""
		subject = nozzle.NewLogSink(labelMaker, logAdapter, newlineToken)
	})

	It("passes fields through to the adapter", func() {
		origin := "cool-origin"
		eventType := events.Envelope_HttpStartStop
		timestamp := int64(time.Now().UnixNano())
		deployment := "neat-deployment"
		job := "some-job"
		index := "an-index"
		ip := "192.168.1.1"
		tags := map[string]string{
			"foo": "bar",
		}

		method := events.Method_GET
		peerType := events.PeerType_Client

		event := events.HttpStartStop{
			Method:   &method,
			PeerType: &peerType,
		}

		envelope := &events.Envelope{
			Origin:        &origin,
			EventType:     &eventType,
			Timestamp:     &timestamp,
			Deployment:    &deployment,
			Job:           &job,
			Index:         &index,
			Ip:            &ip,
			Tags:          tags,
			HttpStartStop: &event,
		}

		subject.Receive(envelope)

		Expect(logAdapter.PostedLogs).To(HaveLen(1))
		postedLog := logAdapter.PostedLogs[0]
		Expect(postedLog.Labels).To(Equal(labels))

		payload := (postedLog.Payload).(map[string]interface{})
		Expect(payload).To(HaveKeyWithValue("eventType", "HttpStartStop"))
		Expect(payload).To(HaveKeyWithValue("deployment", deployment))
		Expect(payload).To(HaveKeyWithValue("job", job))
		Expect(payload).To(HaveKeyWithValue("index", index))
		Expect(payload).To(HaveKeyWithValue("ip", ip))
		Expect(payload).To(HaveKeyWithValue("timestamp", timestamp))
		Expect(payload).To(HaveKey("tags"))
		Expect(payload["tags"].(map[string]interface{})).To(HaveKeyWithValue("foo", "bar"))
	})

	Describe("Payload translation", func() {
		It("handles HttpStartStop", func() {
			method := events.Method_GET
			peerType := events.PeerType_Client
			var low uint64 = 0x7243cc580bc17af4
			var high uint64 = 0x79d4c3b2020e67a5
			requestId := events.UUID{
				Low:  &low,
				High: &high,
			}
			event := events.HttpStartStop{
				Method:    &method,
				PeerType:  &peerType,
				RequestId: &requestId,
			}

			eventType := events.Envelope_HttpStartStop
			envelope := &events.Envelope{
				EventType:     &eventType,
				HttpStartStop: &event,
			}

			subject.Receive(envelope)

			postedLog := logAdapter.PostedLogs[0]
			payload := (postedLog.Payload).(map[string]interface{})
			Expect(payload).To(HaveKeyWithValue("eventType", "HttpStartStop"))
			Expect(payload).To(HaveKey("httpStartStop"))
			Expect(payload).To(HaveKeyWithValue("httpStartStop", map[string]interface{}{
				"method":    "GET",
				"peerType":  "Client",
				"requestId": "f47ac10b-58cc-4372-a567-0e02b2c3d479",
			}))
			Expect(payload).To(HaveKeyWithValue("serviceContext", map[string]interface{}{
				"service": "ab313b25-aa48-4a8f-8e7d-d63a6d410e7c",
			}))
		})

		It("has resolved labels and payloads equivalent for LogMessage", func() {
			eventType := events.Envelope_LogMessage
			messageType := events.LogMessage_OUT

			event := events.LogMessage{
				MessageType: &messageType,
				Message:     []byte("19400: Success: Go"),
			}
			envelope := &events.Envelope{
				EventType:  &eventType,
				LogMessage: &event,
			}

			subject.Receive(envelope)

			postedLog := logAdapter.PostedLogs[0]
			payload := (postedLog.Payload).(map[string]interface{})

			Expect(payload).To(Equal(map[string]interface{}{
				"eventType": eventType.String(),
				"logMessage": map[string]interface{}{
					"message_type": "OUT",
					"message":      "19400: Success: Go",
				},
				"message": "19400: Success: Go",
				"serviceContext": map[string]interface{}{
					"service": "ab313b25-aa48-4a8f-8e7d-d63a6d410e7c",
				},
			}))
			Expect(postedLog.Severity).To(Equal(logging.Default))
		})

		It("has resolved severity for a LogMessage from an Error", func() {
			eventType := events.Envelope_LogMessage
			messageType := events.LogMessage_ERR

			event := events.LogMessage{
				MessageType: &messageType,
			}
			envelope := &events.Envelope{
				EventType:  &eventType,
				LogMessage: &event,
			}

			subject.Receive(envelope)

			postedLog := logAdapter.PostedLogs[0]

			Expect(postedLog.Severity).To(Equal(logging.Error))
		})

		It("has severity and message for Error event types", func() {
			eventType := events.Envelope_Error
			source := "cf-source"
			code := int32(-1)
			message := "some error message"
			event := events.Error{
				Source:  &source,
				Code:    &code,
				Message: &message,
			}
			envelope := &events.Envelope{
				EventType: &eventType,
				Error:     &event,
			}

			subject.Receive(envelope)

			postedLog := logAdapter.PostedLogs[0]

			payload, ok := postedLog.Payload.(map[string]interface{})
			Expect(ok).To(BeTrue())
			Expect(payload["message"]).To(Equal("some error message"))
			Expect(postedLog.Severity).To(Equal(logging.Error))
		})

		It("translates newline tokens when one is passed in", func() {
			subject = nozzle.NewLogSink(labelMaker, logAdapter, "∴")

			eventType := events.Envelope_LogMessage
			messageType := events.LogMessage_OUT

			event := events.LogMessage{
				MessageType: &messageType,
				Message:     []byte("Line one∴  Line two∴  Linethree"),
			}
			envelope := &events.Envelope{
				EventType:  &eventType,
				LogMessage: &event,
			}

			subject.Receive(envelope)

			postedLog := logAdapter.PostedLogs[0]
			payload := (postedLog.Payload).(map[string]interface{})

			expectedMessage := `Line one
  Line two
  Linethree`

			Expect(payload).To(HaveKeyWithValue("message", expectedMessage))
			Expect(payload).To(HaveKeyWithValue("logMessage", map[string]interface{}{
				"message_type": "OUT",
				"message":      expectedMessage,
			},
			))
		})
	})
})
