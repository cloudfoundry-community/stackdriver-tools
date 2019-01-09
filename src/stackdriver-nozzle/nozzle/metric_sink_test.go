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

package nozzle

import (
	"context"
	"errors"
	"time"

	"code.cloudfoundry.org/lager"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/cloudfoundry"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/messages"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/mocks"
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
		subject        Sink
		metricBuffer   *mocks.MetricsBuffer
		unitParser     *mockUnitParser
		logger         *mocks.MockLogger
		labelMaker     LabelMaker
		counterTracker *CounterTracker
		err            error
	)

	BeforeEach(func() {
		appInfoRepository := &mocks.AppInfoRepository{AppInfoMap: map[string]cloudfoundry.AppInfo{}}
		labelMaker = NewLabelMaker(appInfoRepository, "foobar")
		metricBuffer = &mocks.MetricsBuffer{}
		unitParser = &mockUnitParser{}
		logger = &mocks.MockLogger{}

		subject, err = NewMetricSink(logger, "firehose", labelMaker, metricBuffer, counterTracker, unitParser, "^runtimeMetric\\..*")
		Expect(err).To(BeNil())
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
			"Name":      Equal("firehose/origin.valueMetricName"),
			"Labels":    Equal(map[string]string{"foundation": "foobar"}),
			"Value":     Equal(123.456),
			"IntValue":  BeNumerically("==", 0),
			"EventTime": BeTemporally("~", eventTime),
			"StartTime": BeTemporally("~", eventTime),
			"Unit":      Equal("{foo}"),
			"Type":      Equal(eventType),
		}))
		Expect(metrics[0].EventTime.UnixNano()).To(Equal(timeStamp))

		Expect(unitParser.lastInput).To(Equal("barUnit"))
	})

	It("handles runtime ValueMetric", func() {
		eventTime := time.Now()

		origin := "myOrigin"
		name := "runtimeMetric.foobar"
		value := 123.456
		event := events.ValueMetric{
			Name:  &name,
			Value: &value,
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
			"Name":      Equal("firehose/runtimeMetric.foobar"),
			"Labels":    Equal(map[string]string{"foundation": "foobar", "origin": "myOrigin"}),
			"Value":     Equal(123.456),
			"IntValue":  BeNumerically("==", 0),
			"EventTime": BeTemporally("~", eventTime),
			"StartTime": BeTemporally("~", eventTime),
			"Unit":      Ignore(),
			"Type":      Ignore(),
		}))
		Expect(metrics[0].EventTime.UnixNano()).To(Equal(timeStamp))
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
		Expect(metrics).To(HaveLen(5))

		eventName := func(element interface{}) string {
			return element.(messages.Metric).Name
		}

		labels := map[string]string{"foundation": "foobar", "instanceIndex": "0"}
		Expect(metrics).To(MatchAllElements(eventName, Elements{
			"firehose/origin.diskBytesQuota":   MatchFields(IgnoreExtras, Fields{"Labels": Equal(labels), "Type": Equal(metricType), "Value": Equal(float64(1073741824)), "Unit": Equal("")}),
			"firehose/origin.cpuPercentage":    MatchFields(IgnoreExtras, Fields{"Labels": Equal(labels), "Type": Equal(metricType), "Value": Equal(float64(0.061651273460637)), "Unit": Equal("")}),
			"firehose/origin.diskBytes":        MatchFields(IgnoreExtras, Fields{"Labels": Equal(labels), "Type": Equal(metricType), "Value": Equal(float64(164634624)), "Unit": Equal("")}),
			"firehose/origin.memoryBytes":      MatchFields(IgnoreExtras, Fields{"Labels": Equal(labels), "Type": Equal(metricType), "Value": Equal(float64(16601088)), "Unit": Equal("")}),
			"firehose/origin.memoryBytesQuota": MatchFields(IgnoreExtras, Fields{"Labels": Equal(labels), "Type": Equal(metricType), "Value": Equal(float64(33554432)), "Unit": Equal("")}),
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
			"firehose/origin.counterName.delta": MatchAllFields(Fields{
				"Name":      Ignore(),
				"Labels":    Equal(map[string]string{"foundation": "foobar"}),
				"Value":     Equal(float64(654321)),
				"IntValue":  BeNumerically("==", 0),
				"EventTime": BeTemporally("~", eventTime),
				"StartTime": BeTemporally("~", eventTime),
				"Unit":      Equal(""),
				"Type":      Equal(events.Envelope_ValueMetric),
			}),
			"firehose/origin.counterName.total": MatchAllFields(Fields{
				"Name":      Ignore(),
				"Labels":    Equal(map[string]string{"foundation": "foobar"}),
				"Value":     Equal(float64(123456)),
				"IntValue":  BeNumerically("==", 0),
				"EventTime": BeTemporally("~", eventTime),
				"StartTime": BeTemporally("~", eventTime),
				"Unit":      Equal(""),
				"Type":      Equal(events.Envelope_ValueMetric),
			}),
		}))
	})

	Context("with CounterTracker enabled", func() {
		BeforeEach(func() {
			counterTracker = NewCounterTracker(context.TODO(), time.Duration(5)*time.Second, logger)
			subject, err = NewMetricSink(logger, "firehose", labelMaker, metricBuffer, counterTracker, unitParser, "^runtimeMetric\\..*")
			Expect(err).To(BeNil())
		})

		It("creates cumulative metrics for CounterEvent", func() {
			eventTime := time.Now()

			eventType := events.Envelope_CounterEvent
			origin := "origin"
			name := "counterName"

			// List of {delta, total} events to produce.
			eventValues := [][]uint64{
				{5, 105},
				{10, 115},
				{10, 125},
				{5, 5}, // counter reset
				{20, 25},
			}

			for idx, values := range eventValues {
				ts := eventTime.UnixNano() + int64(time.Second)*int64(idx) // Events are 1 second apart.
				delta := values[0]
				total := values[1]
				subject.Receive(&events.Envelope{
					Origin:    &origin,
					EventType: &eventType,
					Timestamp: &ts,
					CounterEvent: &events.CounterEvent{
						Name:  &name,
						Delta: &delta,
						Total: &total,
					},
				})
			}

			metrics := metricBuffer.PostedMetrics
			Expect(metrics).To(HaveLen(4))
			eventName := func(element interface{}) string {
				return element.(messages.Metric).Name
			}
			Expect(metrics).To(MatchElements(eventName, AllowDuplicates, Elements{
				"firehose/origin.counterName": MatchAllFields(Fields{
					"Name":      Ignore(),
					"Labels":    Equal(map[string]string{"foundation": "foobar"}),
					"Value":     BeNumerically("==", 0),
					"IntValue":  Ignore(),
					"EventTime": Ignore(),
					"StartTime": BeTemporally("~", eventTime),
					"Unit":      Equal(""),
					"Type":      Equal(eventType),
				}),
			}))
			expectedTotals := []float64{10, 20, 25, 45}
			for idx, total := range expectedTotals {
				Expect(metrics[idx]).To(MatchFields(IgnoreExtras, Fields{
					"IntValue":  BeNumerically("==", total),
					"EventTime": BeTemporally("~", eventTime.Add(time.Duration(idx+1)*time.Second)),
				}))
			}
		})
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
