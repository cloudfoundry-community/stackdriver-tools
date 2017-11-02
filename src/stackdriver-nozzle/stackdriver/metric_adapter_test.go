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

package stackdriver_test

import (
	"errors"
	"time"

	"sync"

	"fmt"

	"strconv"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/messages"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/mocks"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/stackdriver"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	labelpb "google.golang.org/genproto/googleapis/api/label"
	metricpb "google.golang.org/genproto/googleapis/api/metric"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

const batchSize = 200

var _ = Describe("MetricAdapter", func() {
	var (
		subject     stackdriver.MetricAdapter
		client      *mocks.MockClient
		heartbeater *mocks.Heartbeater
	)

	BeforeEach(func() {
		client = &mocks.MockClient{}
		heartbeater = mocks.NewHeartbeater()
		subject, _ = stackdriver.NewMetricAdapter("my-awesome-project", client, batchSize, heartbeater)
	})

	It("takes metrics and posts a time series", func() {
		eventTime := time.Now()

		metrics := []*messages.Metric{
			{
				Name:      "metricName",
				Value:     123.45,
				EventTime: eventTime,
			},
			{
				Name:      "secondMetricName",
				Value:     54.321,
				EventTime: eventTime,
			},
		}
		labels := map[string]string{
			"key": "name",
		}

		metricEvents := []*messages.MetricEvent{{Labels: labels, Metrics: metrics}}

		subject.PostMetricEvents(metricEvents)

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

	// TODO(jrjohnson): This should be a table test
	It("posts in the correct batch size", func() {
		postCount := 0
		timeSeriesCount := 0
		client.PostFn = func(req *monitoringpb.CreateTimeSeriesRequest) error {
			postCount += 1
			timeSeriesCount += len(req.TimeSeries)

			if len(req.TimeSeries) > batchSize {
				Fail(fmt.Sprintf("time series (%d) exceeds batch size (%d)", len(req.TimeSeries), batchSize))
			}

			return nil
		}

		metricGroupSizes := []int{199, 200, 201, 399, 400, 1999, 2000, 2001}
		// Test various numbers of metrics being posted to the buffer
		for _, groupSize := range metricGroupSizes {
			events := []*messages.MetricEvent{}

			for i := 0; i < groupSize; i++ {
				events = append(events, &messages.MetricEvent{
					Labels:  map[string]string{"Name": strconv.Itoa(i)},
					Metrics: []*messages.Metric{{Value: 1}, {Value: 2}},
				})
			}

			subject.PostMetricEvents(events)

			Expect(postCount).To(BeNumerically("<", groupSize))
			Expect(timeSeriesCount).To(Equal(groupSize * 2))

			postCount = 0
			timeSeriesCount = 0
		}
	})

	It("creates metric descriptors", func() {
		labels := map[string]string{"key": "value"}

		metrics := []*messages.Metric{
			{
				Name: "metricWithUnit",
				Unit: "{foobar}",
			},
			{
				Name: "metricWithoutUnit",
			},
		}
		metricEvents := []*messages.MetricEvent{{Labels: labels, Metrics: metrics}}

		subject.PostMetricEvents(metricEvents)

		Expect(client.DescriptorReqs).To(HaveLen(1))
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
		metricEvents := []*messages.MetricEvent{{Metrics: metrics}}

		subject.PostMetricEvents(metricEvents)

		Expect(client.DescriptorReqs).To(HaveLen(2))
	})

	It("handles concurrent metric descriptor creation", func() {
		metricEventFromName := func(name string) []*messages.MetricEvent {
			return []*messages.MetricEvent{{Metrics: []*messages.Metric{
				{
					Name: name,
					Unit: "{foobar}",
				},
			}}}
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

		go subject.PostMetricEvents(metricEventFromName("a"))
		go subject.PostMetricEvents(metricEventFromName("b"))
		go subject.PostMetricEvents(metricEventFromName("a"))
		go subject.PostMetricEvents(metricEventFromName("c"))
		go subject.PostMetricEvents(metricEventFromName("b"))

		Eventually(func() int {
			mutex.Lock()
			defer mutex.Unlock()

			return callCount
		}).Should(Equal(3))
	})

	It("returns the adapter even if we fail to list the metric descriptors", func() {
		expectedErr := errors.New("fail")
		client.ListErr = expectedErr
		subject, err := stackdriver.NewMetricAdapter("my-awesome-project", client, 1, heartbeater)
		Expect(subject).To(Not(BeNil()))
		Expect(err).To(Equal(expectedErr))
	})

	It("increments metrics counters", func() {
		metricEvents := []*messages.MetricEvent{
			{Metrics: []*messages.Metric{
				{
					Name: "metricWithUnit",
					Unit: "{foobar}",
				},
				{
					Name: "metricWithUnitToo",
					Unit: "{barfoo}",
				}}},
			{Metrics: []*messages.Metric{
				{
					Name: "anExistingMetric",
					Unit: "{lalala}",
				},
			}}}

		Expect(subject.PostMetricEvents(metricEvents)).To(Succeed())
		Expect(heartbeater.GetCount("metrics.events.count")).To(Equal(2))
		Expect(heartbeater.GetCount("metrics.timeseries.count")).To(Equal(3))
		Expect(heartbeater.GetCount("metrics.requests")).To(Equal(1))

		Expect(subject.PostMetricEvents(metricEvents)).To(Succeed())
		Expect(heartbeater.GetCount("metrics.events.count")).To(Equal(4))
		Expect(heartbeater.GetCount("metrics.timeseries.count")).To(Equal(6))
		Expect(heartbeater.GetCount("metrics.requests")).To(Equal(2))
	})

	It("measures out of order errors", func() {
		metricEvents := []*messages.MetricEvent{{Metrics: []*messages.Metric{{}}}}

		client.PostFn = func(req *monitoringpb.CreateTimeSeriesRequest) error {
			return errors.New("GRPC Stuff. Points must be written in order. Other stuff")
		}

		Expect(subject.PostMetricEvents(metricEvents)).To(Succeed())
		Expect(heartbeater.GetCount("metrics.post.errors")).To(Equal(1))
		Expect(heartbeater.GetCount("metrics.post.errors.out_of_order")).To(Equal(1))
		Expect(heartbeater.GetCount("metrics.post.errors.unknown")).To(Equal(0))
	})

	It("measures unknown errors", func() {
		metricEvents := []*messages.MetricEvent{{Metrics: []*messages.Metric{{}}}}

		client.PostFn = func(req *monitoringpb.CreateTimeSeriesRequest) error {
			return errors.New("tragedy strikes")
		}
		Expect(subject.PostMetricEvents(metricEvents)).NotTo(Succeed())
		Expect(heartbeater.GetCount("metrics.post.errors")).To(Equal(1))
		Expect(heartbeater.GetCount("metrics.post.errors.out_of_order")).To(Equal(0))
		Expect(heartbeater.GetCount("metrics.post.errors.unknown")).To(Equal(1))
	})
})
