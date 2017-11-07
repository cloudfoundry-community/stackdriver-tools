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
	"errors"
	"time"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/messages"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/mocks"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/nozzle"
	"github.com/cloudfoundry/lager"
	"github.com/cloudfoundry/sonde-go/events"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	. "github.com/onsi/gomega/gstruct"
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
		logger       *mocks.MockLogger
	)

	BeforeEach(func() {
		labels = map[string]string{"foo": "bar"}
		labelMaker := &mocks.LabelMaker{Labels: labels}
		metricBuffer = &mocks.MetricsBuffer{}
		unitParser = &mockUnitParser{}
		logger = &mocks.MockLogger{}

		subject = nozzle.NewMetricSink(logger, labelMaker, metricBuffer, unitParser)
	})

	It("creates metric for ValueMetric", func() {
		eventTime := time.Now()

		origin := "origin"
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
			Origin:      &origin,
			EventType:   &eventType,
			ValueMetric: &event,
			Timestamp:   &timeStamp,
		}

		subject.Receive(envelope)

		metrics := metricBuffer.PostedMetrics
		Expect(metrics).To(HaveLen(1))
		Expect(metrics[0]).To(MatchAllFields(Fields{
			"Name":      Equal("origin.valueMetricName"),
			"Value":     Equal(123.456),
			"EventTime": Ignore(),
			"Unit":      Equal("{foo}"),
		}))
		Expect(metrics[0].EventTime.UnixNano()).To(Equal(timeStamp))

		Expect(unitParser.lastInput).To(Equal("barUnit"))
	})

	It("creates the proper metrics for ContainerMetric", func() {
		eventTime := time.Now()

		origin := "origin"
		diskBytesQuota := uint64(1073741824)
		instanceIndex := int32(0)
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
			Origin:          &origin,
			EventType:       &metricType,
			ContainerMetric: &containerMetric,
			Timestamp:       &timeStamp,
		}

		subject.Receive(envelope)

		metrics := metricBuffer.PostedMetrics
		Expect(metrics).To(HaveLen(6))

		eventName := func(element interface{}) string {
			return element.(messages.Metric).Name
		}

		Expect(metrics).To(MatchAllElements(eventName, Elements{
			"origin.diskBytesQuota":   MatchAllFields(Fields{"Name": Ignore(), "Value": Equal(float64(1073741824)), "EventTime": Ignore(), "Unit": Equal("")}),
			"origin.instanceIndex":    MatchAllFields(Fields{"Name": Ignore(), "Value": Equal(float64(0)), "EventTime": Ignore(), "Unit": Equal("")}),
			"origin.cpuPercentage":    MatchAllFields(Fields{"Name": Ignore(), "Value": Equal(float64(0.061651273460637)), "EventTime": Ignore(), "Unit": Equal("")}),
			"origin.diskBytes":        MatchAllFields(Fields{"Name": Ignore(), "Value": Equal(float64(164634624)), "EventTime": Ignore(), "Unit": Equal("")}),
			"origin.memoryBytes":      MatchAllFields(Fields{"Name": Ignore(), "Value": Equal(float64(16601088)), "EventTime": Ignore(), "Unit": Equal("")}),
			"origin.memoryBytesQuota": MatchAllFields(Fields{"Name": Ignore(), "Value": Equal(float64(33554432)), "EventTime": Ignore(), "Unit": Equal("")}),
		}))
	})

	It("creates total and delta metrics for CounterEvent", func() {
		eventTime := time.Now()

		eventType := events.Envelope_CounterEvent
		origin := "origin"
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
			Origin:       &origin,
			EventType:    &eventType,
			CounterEvent: &event,
			Timestamp:    &timeStamp,
		}

		subject.Receive(envelope)

		metrics := metricBuffer.PostedMetrics

		eventName := func(element interface{}) string {
			return element.(messages.Metric).Name
		}
		Expect(metrics).To(MatchAllElements(eventName, Elements{
			"origin.counterName.delta": MatchAllFields(Fields{
				"Name":      Ignore(),
				"Value":     Equal(float64(654321)),
				"EventTime": Ignore(),
				"Unit":      Equal(""),
			}),
			"origin.counterName.total": MatchAllFields(Fields{
				"Name":      Ignore(),
				"Value":     Equal(float64(123456)),
				"EventTime": Ignore(),
				"Unit":      Equal(""),
			}),
		}))
	})

	It("returns error when envelope contains unhandled event type", func() {
		eventType := events.Envelope_HttpStartStop
		envelope := &events.Envelope{
			EventType: &eventType,
		}

		subject.Receive(envelope)

		Expect(logger.Logs()).To(ContainElement(mocks.Log{
			Action: "metricSink.Receive",
			Level:  lager.ERROR,
			Err:    errors.New("unknown event type: HttpStartStop"),
		}))
	})
})
