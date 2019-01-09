/*
 * Copyright 2019 Google Inc.
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

package metricspipeline

import (
	"time"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/messages"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/mocks"
	"github.com/cloudfoundry/sonde-go/events"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Router", func() {
	var (
		metricAdapter *mocks.MetricAdapter
		logAdapter    *mocks.LogAdapter
	)
	BeforeEach(func() {
		metricAdapter = &mocks.MetricAdapter{}
		logAdapter = &mocks.LogAdapter{}
	})
	It("can route events to a single location", func() {
		metricEvent := events.Envelope_ContainerMetric
		logEvent := events.Envelope_ValueMetric

		router := NewRouter(metricAdapter, []events.Envelope_EventType{metricEvent}, logAdapter, []events.Envelope_EventType{logEvent})
		router.PostMetrics([]*messages.Metric{
			{Type: metricEvent},
			{Type: logEvent},
		})

		Expect(metricAdapter.PostedMetrics).To(HaveLen(1))
		Expect(metricAdapter.PostMetricsCount).To(Equal(1))
		Expect(metricAdapter.PostedMetrics[0].Type).To(Equal(metricEvent))
		Expect(logAdapter.PostedLogs).To(HaveLen(1))
		Expect(logAdapter.PostedLogs[0].Payload.(*messages.Metric).Type).To(Equal(logEvent))
	})

	It("can route an event to two locations", func() {
		metricEvent := events.Envelope_ContainerMetric
		logEvent := events.Envelope_ValueMetric
		events := []events.Envelope_EventType{metricEvent, logEvent}

		router := NewRouter(metricAdapter, events, logAdapter, events)
		router.PostMetrics([]*messages.Metric{
			{Type: metricEvent},
			{Type: logEvent},
		})

		Expect(metricAdapter.PostedMetrics).To(HaveLen(2))
		Expect(metricAdapter.PostMetricsCount).To(Equal(1))
		Expect(metricAdapter.PostedMetrics[0].Type).To(Equal(metricEvent))
		Expect(metricAdapter.PostedMetrics[1].Type).To(Equal(logEvent))
		Expect(logAdapter.PostedLogs).To(HaveLen(2))
		Expect(logAdapter.PostedLogs[0].Payload.(*messages.Metric).Type).To(Equal(metricEvent))
		Expect(logAdapter.PostedLogs[1].Payload.(*messages.Metric).Type).To(Equal(logEvent))
	})

	It("can translate Metric statements to Logs", func() {
		logEvent := events.Envelope_ValueMetric
		labels := map[string]string{"foo": "bar"}
		metric := &messages.Metric{
			Name:      "valueMetric",
			Labels:    labels,
			Value:     float64(123),
			EventTime: time.Now(),
			Unit:      "f",
			Type:      logEvent,
		}
		router := NewRouter(nil, nil, logAdapter, []events.Envelope_EventType{logEvent})
		router.PostMetrics([]*messages.Metric{metric})
		Expect(logAdapter.PostedLogs).To(HaveLen(1))
		log := logAdapter.PostedLogs[0]
		Expect(log.Labels).To(Equal(labels))
		Expect(log.Payload).To(BeAssignableToTypeOf(&messages.Metric{}))
		payload := log.Payload.(*messages.Metric)
		Expect(payload).To(Equal(metric))
		Expect(payload.Type).To(Equal(logEvent))
		Expect(payload.Labels).To(Equal(labels))
	})
})
