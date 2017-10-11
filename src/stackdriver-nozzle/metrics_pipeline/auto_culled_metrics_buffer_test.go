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

package metrics_pipeline_test

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"time"

	"sort"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/messages"
	. "github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/metrics_pipeline"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/mocks"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("autoCulledMetricsBuffer", func() {
	var (
		metricAdapter *mocks.MetricAdapter
		heartbeater   *mocks.Heartbeater
		logger        *mocks.MockLogger
	)

	BeforeEach(func() {
		metricAdapter = &mocks.MetricAdapter{}
		heartbeater = mocks.NewHeartbeater()
		logger = &mocks.MockLogger{}
	})

	It("culls duplicate metrics", func() {
		subject, _ := NewAutoCulledMetricsBuffer(context.TODO(), logger, 100*time.Millisecond, 5, metricAdapter, heartbeater)

		subject.PostMetricEvents([]*messages.MetricEvent{
			{
				Labels:  map[string]string{"Name": "a"},
				Metrics: []*messages.Metric{{Name: "a", Value: 1}},
			},
			{
				Labels:  map[string]string{"Name": "a"},
				Metrics: []*messages.Metric{{Name: "a", Value: 2}, {Name: "b", Value: 2}},
			},
		})
		Eventually(metricAdapter.GetPostedMetricEvents).Should(HaveLen(1))

		expected := []*messages.MetricEvent{{
			Labels:  map[string]string{"Name": "a"},
			Metrics: []*messages.Metric{{Name: "a", Value: 2}, {Name: "b", Value: 2}},
		}}
		postedEvent := metricAdapter.GetPostedMetricEvents()[0]
		Expect(postedEvent.Metrics).To(HaveLen(2))
		Expect(postedEvent).To(BeEquivalentTo(expected[0]))
		Expect(heartbeater.GetCount("metrics.events.sampled")).To(Equal(1))
	})

	It("culls multiple duplicates", func() {
		subject, _ := NewAutoCulledMetricsBuffer(context.TODO(), logger, 100*time.Millisecond, 5, metricAdapter, heartbeater)

		subject.PostMetricEvents([]*messages.MetricEvent{
			{
				Labels:  map[string]string{"d1": "a"},
				Metrics: []*messages.Metric{{Value: 1}},
			},
			{
				Labels:  map[string]string{"d2": "a"},
				Metrics: []*messages.Metric{{Value: 2}},
			},
			{
				Labels:  map[string]string{"d3": "a"},
				Metrics: []*messages.Metric{{Value: 3}},
			},
			{
				Labels:  map[string]string{"d3": "a"},
				Metrics: []*messages.Metric{{Value: 4}},
			},
			{
				Labels:  map[string]string{"d3": "a"},
				Metrics: []*messages.Metric{{Value: 5}},
			},
		})

		Eventually(metricAdapter.GetPostedMetricEvents).Should(HaveLen(3))
		expected := sortableMetrics{
			{
				Labels:  map[string]string{"d1": "a"},
				Metrics: []*messages.Metric{{Value: 1}},
			},
			{
				Labels:  map[string]string{"d2": "a"},
				Metrics: []*messages.Metric{{Value: 2}},
			},
			{
				Labels:  map[string]string{"d3": "a"},
				Metrics: []*messages.Metric{{Value: 5}},
			},
		}
		actual := sortableMetrics(metricAdapter.GetPostedMetricEvents())

		sort.Sort(expected)
		sort.Sort(actual)

		Expect(actual).To(BeEquivalentTo(expected))
		Expect(heartbeater.GetCount("metrics.events.sampled")).To(Equal(2))
	})

	It("it buffers metrics for the expected duration before flushing", func() {
		d := 500 * time.Millisecond
		subject, _ := NewAutoCulledMetricsBuffer(context.TODO(), logger, d, 5, metricAdapter, heartbeater)

		subject.PostMetricEvents([]*messages.MetricEvent{
			{
				Labels:  map[string]string{"Name": "a"},
				Metrics: []*messages.Metric{{Value: 1}},
			},
			{
				Labels:  map[string]string{"Name": "b"},
				Metrics: []*messages.Metric{{Value: 2}},
			},
		})
		Expect(metricAdapter.GetPostedMetricEvents()).Should(HaveLen(0))
		Eventually(metricAdapter.GetPostedMetricEvents).Should(HaveLen(2))
	})

	It("it flushes metrics when the context is canceled", func() {
		d := 500 * time.Second
		ctx, cancel := context.WithCancel(context.Background())
		subject, _ := NewAutoCulledMetricsBuffer(ctx, logger, d, 5, metricAdapter, heartbeater)

		subject.PostMetricEvents([]*messages.MetricEvent{
			{
				Labels:  map[string]string{"Name": "a"},
				Metrics: []*messages.Metric{{Value: 1}},
			},
			{
				Labels:  map[string]string{"Name": "b"},
				Metrics: []*messages.Metric{{Value: 2}},
			},
		})
		cancel()
		Eventually(metricAdapter.GetPostedMetricEvents).Should(HaveLen(2))
	})

	It("it posts the metrics in correct batch size", func() {
		d := 10 * time.Millisecond
		batchSize := 200

		metricAdapter.PostMetricEventsFn = func(metricEvents []*messages.MetricEvent) error {
			if len(metricEvents) > batchSize {
				return fmt.Errorf("Batch size (%v) exceeded max (%v)", len(metricEvents), batchSize)
			}

			metricAdapter.PostedMetricEvents = append(metricAdapter.PostedMetricEvents, metricEvents...)
			return metricAdapter.PostMetricEventsError
		}

		metricGroupSizes := []int{199, 200, 201, 399, 400, 1999, 2000, 2001}

		// Test various numbers of metrics being posted to the buffer
		for _, groupSize := range metricGroupSizes {
			ctx, cancel := context.WithCancel(context.Background())
			metricAdapter.PostedMetricEvents = []*messages.MetricEvent{}
			metricAdapter.PostMetricEventsError = nil
			subject, errs := NewAutoCulledMetricsBuffer(ctx, logger, d, batchSize, metricAdapter, heartbeater)
			for i := 0; i < groupSize; i++ {
				subject.PostMetricEvents([]*messages.MetricEvent{
					{
						Labels:  map[string]string{"Name": strconv.Itoa(i)},
						Metrics: []*messages.Metric{{Value: 1}},
					},
				})
			}
			cancel()
			err := <-errs
			Expect(err).ToNot(HaveOccurred())
			Expect(metricAdapter.PostedMetricEvents).To(HaveLen(groupSize))
		}

	})

	It("sends errors through the error channel", func() {
		d := 1 * time.Millisecond
		subject, errs := NewAutoCulledMetricsBuffer(context.TODO(), logger, d, 5, metricAdapter, heartbeater)

		expectedErr := errors.New("fail")
		metricAdapter.PostMetricEventsError = expectedErr

		metricEvent := []*messages.MetricEvent{{}}
		subject.PostMetricEvents(metricEvent)

		Eventually(metricAdapter.GetPostedMetricEvents).Should(HaveLen(1))

		var err error
		Eventually(errs).Should(Receive(&err))
		Expect(err).To(Equal(expectedErr))
	})

	Describe("with a slow MetricAdapter", func() {
		var (
			metricPosted chan interface{}
			subject      MetricsBuffer
		)

		BeforeEach(func() {
			metricPosted = make(chan interface{})
			metricAdapter.PostMetricEventsFn = func([]*messages.MetricEvent) error {
				metricPosted <- struct{}{}
				time.Sleep(30 * time.Second)
				return nil
			}

			subject, _ = NewAutoCulledMetricsBuffer(context.TODO(), logger, 1*time.Millisecond, 5, metricAdapter, heartbeater)
		})

		It("doesn't block new metrics during flush", func() {
			metric := []*messages.MetricEvent{{}}
			subject.PostMetricEvents(metric)

			Eventually(metricPosted).Should(Receive())
			unblocked := make(chan interface{})
			go func() {
				subject.PostMetricEvents(metric)
				unblocked <- struct{}{}
			}()
			Eventually(unblocked).Should(Receive())
		})
	})
})

type sortableMetrics []*messages.MetricEvent

func (b sortableMetrics) Len() int {
	return len(b)
}
func (b sortableMetrics) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}
func (b sortableMetrics) Less(i, j int) bool {
	return b[i].Hash() < b[j].Hash()
}
