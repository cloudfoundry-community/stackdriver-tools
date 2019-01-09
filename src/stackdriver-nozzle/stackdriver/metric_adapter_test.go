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

package stackdriver

import (
	"errors"
	"strconv"
	"sync"
	"time"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/messages"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/mocks"
	"github.com/cloudfoundry/sonde-go/events"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/ginkgo/extensions/table"
	. "github.com/onsi/gomega"
	labelpb "google.golang.org/genproto/googleapis/api/label"
	metricpb "google.golang.org/genproto/googleapis/api/metric"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

const batchSize = 200

var _ = Describe("MetricAdapter", func() {
	var (
		subject MetricAdapter
		client  *mocks.MockClient
		logger  *mocks.MockLogger
	)

	BeforeEach(func() {
		timeSeriesCount.Set(0)

		client = &mocks.MockClient{}
		logger = &mocks.MockLogger{}
		subject, _ = NewMetricAdapter("my-awesome-project", client, batchSize, logger)
	})

	It("takes metrics and posts a time series", func() {
		eventTime := time.Now()

		labels := map[string]string{
			"key": "name",
		}
		metrics := []*messages.Metric{
			{
				Name:      "metricName",
				Labels:    labels,
				Value:     123.45,
				EventTime: eventTime,
			},
			{
				Name:      "secondMetricName",
				Labels:    labels,
				Value:     54.321,
				EventTime: eventTime,
			},
		}

		subject.PostMetrics(metrics)

		Expect(client.MetricReqs).To(HaveLen(1))

		req := client.MetricReqs[0]
		Expect(req.Name).To(Equal("projects/my-awesome-project"))

		timeSerieses := req.GetTimeSeries()
		Expect(timeSerieses).To(HaveLen(len(metrics)))

		timeSeries := timeSerieses[0]
		Expect(timeSeries.GetMetric().Type).To(Equal("custom.googleapis.com/metricName"))
		Expect(timeSeries.GetMetric().Labels).To(Equal(labels))
		Expect(timeSeries.GetPoints()).To(HaveLen(1))

		point := timeSeries.GetPoints()[0]
		Expect(point.GetInterval().GetEndTime().Seconds).To(Equal(int64(eventTime.Unix())))
		Expect(point.GetInterval().GetEndTime().Nanos).To(Equal(int32(eventTime.Nanosecond())))
		value, ok := point.GetValue().GetValue().(*monitoringpb.TypedValue_DoubleValue)
		Expect(ok).To(BeTrue())
		Expect(value.DoubleValue).To(Equal(123.45))

		timeSeries = timeSerieses[1]
		Expect(timeSeries.GetMetric().Type).To(Equal("custom.googleapis.com/secondMetricName"))
		Expect(timeSeries.GetMetric().Labels).To(Equal(labels))
		Expect(timeSeries.GetPoints()).To(HaveLen(1))

		point = timeSeries.GetPoints()[0]
		value, ok = point.GetValue().GetValue().(*monitoringpb.TypedValue_DoubleValue)
		Expect(ok).To(BeTrue())
		Expect(value.DoubleValue).To(Equal(54.321))
	})

	type postMetrics struct {
		metricCount int
		postCount   int
	}

	DescribeTable("correct batch size",
		func(t postMetrics) {
			metrics := []*messages.Metric{}
			for i := 0; i < t.metricCount; i++ {
				metrics = append(metrics, &messages.Metric{
					Labels: map[string]string{"Name": strconv.Itoa(i)},
					Value:  float64(i),
				})
			}
			subject.PostMetrics(metrics)

			Expect(client.MetricReqs).To(HaveLen(t.postCount))
			Expect(client.TimeSeries).To(HaveLen(t.metricCount))
		},
		Entry("less than the batch size", postMetrics{1, 1}),
		Entry("exactly the batch size", postMetrics{200, 1}),
		Entry("one over the batch size", postMetrics{201, 2}),
		Entry("a large batch size", postMetrics{4001, 21}))

	It("creates metric descriptors", func() {
		labels := map[string]string{"key": "value"}

		metrics := []*messages.Metric{
			{
				Name:   "metricWithUnit",
				Labels: labels,
				Unit:   "{foobar}",
			},
			{
				Name:   "metricWithoutUnit",
				Labels: labels,
			},
			{
				Name:   "someCounter",
				Labels: labels,
				Type:   events.Envelope_CounterEvent,
			},
		}

		subject.PostMetrics(metrics)

		Expect(client.DescriptorReqs).To(HaveLen(2))
		req := client.DescriptorReqs[0]
		Expect(req.Name).To(Equal("projects/my-awesome-project"))
		Expect(req.MetricDescriptor).To(Equal(&metricpb.MetricDescriptor{
			Name:        "projects/my-awesome-project/metricDescriptors/custom.googleapis.com/metricWithUnit",
			Type:        "custom.googleapis.com/metricWithUnit",
			Labels:      []*labelpb.LabelDescriptor{{Key: "key", ValueType: 0, Description: ""}},
			MetricKind:  metricpb.MetricDescriptor_GAUGE,
			ValueType:   metricpb.MetricDescriptor_DOUBLE,
			Unit:        "{foobar}",
			Description: "stackdriver-nozzle created custom metric.",
			DisplayName: "metricWithUnit",
		}))
		Expect(client.DescriptorReqs[1].MetricDescriptor.MetricKind).To(Equal(metricpb.MetricDescriptor_CUMULATIVE))
		Expect(client.DescriptorReqs[1].MetricDescriptor.ValueType).To(Equal(metricpb.MetricDescriptor_INT64))
	})

	It("only creates the same descriptor once", func() {
		metrics := []*messages.Metric{
			{
				Name: "metricWithUnit",
				Unit: "{foobar}",
			},
			{
				Name: "metricWithUnitToo",
				Unit: "{barfoo}",
			},
			{
				Name: "metricWithUnit",
				Unit: "{foobar}",
			},
			{
				Name: "anExistingMetric",
				Unit: "{lalala}",
			},
		}
		subject.PostMetrics(metrics)

		Expect(client.DescriptorReqs).To(HaveLen(2))
	})

	It("handles concurrent metric descriptor creation", func() {
		metricEventFromName := func(name string) []*messages.Metric {
			return []*messages.Metric{
				{
					Name: name,
					Unit: "{foobar}",
				},
			}
		}

		var mutex sync.Mutex
		callCount := 0
		client.CreateMetricDescriptorFn = func(request *monitoringpb.CreateMetricDescriptorRequest) error {
			mutex.Lock()
			callCount += 1
			mutex.Unlock()

			time.Sleep(100 * time.Millisecond)
			return nil
		}

		go subject.PostMetrics(metricEventFromName("a"))
		go subject.PostMetrics(metricEventFromName("b"))
		go subject.PostMetrics(metricEventFromName("a"))
		go subject.PostMetrics(metricEventFromName("c"))
		go subject.PostMetrics(metricEventFromName("b"))

		Eventually(func() int {
			mutex.Lock()
			defer mutex.Unlock()

			return callCount
		}).Should(Equal(3))
	})

	It("returns the adapter even if we fail to list the metric descriptors", func() {
		expectedErr := errors.New("fail")
		client.ListErr = expectedErr
		subject, err := NewMetricAdapter("my-awesome-project", client, 1, logger)
		Expect(subject).To(Not(BeNil()))
		Expect(err).To(Equal(expectedErr))
	})

	It("increments metrics counters", func() {
		metrics := []*messages.Metric{
			{
				Name: "metricWithUnit",
				Unit: "{foobar}",
			},
			{
				Name: "metricWithUnitToo",
				Unit: "{barfoo}",
			},
			{
				Name: "anExistingMetric",
				Unit: "{lalala}",
			},
		}

		subject.PostMetrics(metrics)
		Expect(timeSeriesCount.IntValue()).To(Equal(3))

		subject.PostMetrics(metrics)
		Expect(timeSeriesCount.IntValue()).To(Equal(6))
	})
})
