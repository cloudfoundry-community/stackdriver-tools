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

	"fmt"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/mocks"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/nozzle"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/stackdriver"
	"github.com/cloudfoundry/sonde-go/events"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type mockUnitParser struct {
	lastInput string
}

func (m *mockUnitParser) Parse(unit string) string {
	m.lastInput = unit
	return "{foo}"
}

var _ = Describe("MetricSink", func() {
	var (
		subject      nozzle.Sink
		metricBuffer *mocks.MetricsBuffer
		unitParser   *mockUnitParser
		labels       map[string]string
	)

	BeforeEach(func() {
		labels = map[string]string{"foo": "bar"}
		labelMaker := &mocks.LabelMaker{Labels: labels}
		metricBuffer = &mocks.MetricsBuffer{}
		unitParser = &mockUnitParser{}

		subject = nozzle.NewMetricSink(labelMaker, metricBuffer, unitParser)
	})

	It("creates metric for ValueMetric", func() {
		eventTime := time.Now()

		name := "valueMetricName"
		value := 123.456
		unit := "barUnit"
		event := events.ValueMetric{
			Name:  &name,
			Value: &value,
			Unit:  &unit,
		}

		eventType := events.Envelope_ValueMetric
		timeStamp := eventTime.UnixNano()
		envelope := &events.Envelope{
			EventType:   &eventType,
			ValueMetric: &event,
			Timestamp:   &timeStamp,
		}

		err := subject.Receive(envelope)
		Expect(err).To(BeNil())

		metrics := metricBuffer.PostedMetrics
		Expect(metrics).To(ConsistOf(stackdriver.Metric{
			Name:      "valueMetricName",
			Value:     123.456,
			Labels:    labels,
			EventTime: eventTime,
			Unit:      "{foo}",
		}))

		Expect(unitParser.lastInput).To(Equal("barUnit"))
	})

	It("creates the proper metrics for ContainerMetric", func() {
		eventTime := time.Now()

		diskBytesQuota := uint64(1073741824)
		instanceIndex := int32(3)
		cpuPercentage := 0.061651273460637
		diskBytes := uint64(164634624)
		memoryBytes := uint64(16601088)
		memoryBytesQuota := uint64(33554432)
		applicationId := "ee2aa52e-3c8a-4851-b505-0cb9fe24806e"
		timeStamp := eventTime.UnixNano()

		metricType := events.Envelope_ContainerMetric
		containerMetric := events.ContainerMetric{
			DiskBytesQuota:   &diskBytesQuota,
			InstanceIndex:    &instanceIndex,
			CpuPercentage:    &cpuPercentage,
			DiskBytes:        &diskBytes,
			MemoryBytes:      &memoryBytes,
			MemoryBytesQuota: &memoryBytesQuota,
			ApplicationId:    &applicationId,
		}

		envelope := &events.Envelope{
			EventType:       &metricType,
			ContainerMetric: &containerMetric,
			Timestamp:       &timeStamp,
		}

		err := subject.Receive(envelope)
		Expect(err).To(BeNil())

		metrics := metricBuffer.PostedMetrics
		Expect(metrics).To(HaveLen(5))

		// ContainerMetric has special labels to specify the specific container
		expectedLabels := map[string]string{}
		for k, v := range labels {
			expectedLabels[k] = v
		}
		expectedLabels["instanceIndex"] = fmt.Sprintf("%v", instanceIndex)

		Expect(metrics).To(ContainElement(stackdriver.Metric{Name: "diskBytesQuota", Value: float64(1073741824), Labels: expectedLabels, EventTime: eventTime, Unit: ""}))
		Expect(metrics).To(ContainElement(stackdriver.Metric{Name: "cpuPercentage", Value: 0.061651273460637, Labels: expectedLabels, EventTime: eventTime, Unit: ""}))
		Expect(metrics).To(ContainElement(stackdriver.Metric{Name: "diskBytes", Value: float64(164634624), Labels: expectedLabels, EventTime: eventTime, Unit: ""}))
		Expect(metrics).To(ContainElement(stackdriver.Metric{Name: "memoryBytes", Value: float64(16601088), Labels: expectedLabels, EventTime: eventTime, Unit: ""}))
		Expect(metrics).To(ContainElement(stackdriver.Metric{Name: "memoryBytesQuota", Value: float64(33554432), Labels: expectedLabels, EventTime: eventTime, Unit: ""}))
	})

	It("creates total and delta metrics for CounterEvent", func() {
		eventTime := time.Now()

		eventType := events.Envelope_CounterEvent
		name := "counterName"
		delta := uint64(654321)
		total := uint64(123456)
		timeStamp := eventTime.UnixNano()

		event := events.CounterEvent{
			Name:  &name,
			Delta: &delta,
			Total: &total,
		}
		envelope := &events.Envelope{
			EventType:    &eventType,
			CounterEvent: &event,
			Timestamp:    &timeStamp,
		}

		err := subject.Receive(envelope)
		Expect(err).To(BeNil())

		metrics := metricBuffer.PostedMetrics
		Expect(metrics).To(ConsistOf(
			stackdriver.Metric{
				Name:      "counterName.delta",
				Value:     float64(654321),
				Labels:    labels,
				EventTime: eventTime,
				Unit:      "",
			},
			stackdriver.Metric{
				Name:      "counterName.total",
				Value:     float64(123456),
				Labels:    labels,
				EventTime: eventTime,
				Unit:      "",
			},
		))
	})

	It("returns error when envelope contains unhandled event type", func() {
		eventType := events.Envelope_HttpStart
		envelope := &events.Envelope{
			EventType: &eventType,
		}

		err := subject.Receive(envelope)

		Expect(err).NotTo(BeNil())
	})
})
