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
		subject, _ := NewAutoCulledMetricsBuffer(context.TODO(), logger, 100*time.Millisecond, metricAdapter, heartbeater)

		subject.PostMetricEvents([]*messages.MetricEvent{
			{
				Labels: map[string]string{"Name": "a"},
				Metrics: []*messages.Metric{
					{Name: "a", Value: 1, EventTime: time.Unix(1234567890, 0)},
					{Name: "b", Value: 0, EventTime: time.Unix(1234567890, 0)},
				},
			},
			{
				Labels: map[string]string{"Name": "a"},
				Metrics: []*messages.Metric{
					{Name: "a", Value: 2, EventTime: time.Unix(1234567891, 0)},
					{Name: "b", Value: 2, EventTime: time.Unix(1234567891, 0)},
				},
			},
		})
		Eventually(metricAdapter.GetPostedMetricEvents).Should(HaveLen(1))

		expected := []*messages.MetricEvent{{
			Labels: map[string]string{"Name": "a"},
			Metrics: []*messages.Metric{
				{Name: "a", Value: 2, EventTime: time.Unix(1234567891, 0)},
				{Name: "b", Value: 2, EventTime: time.Unix(1234567891, 0)},
			},
		}}
		postedEvent := metricAdapter.GetPostedMetricEvents()[0]
		Expect(postedEvent.Metrics).To(HaveLen(2))
		Expect(postedEvent).To(BeEquivalentTo(expected[0]))
		Expect(heartbeater.GetCount("metrics.events.sampled")).To(Equal(1))
	})

	It("culls multiple duplicates, keeping the latest", func() {
		subject, _ := NewAutoCulledMetricsBuffer(context.TODO(), logger, 100*time.Millisecond, metricAdapter, heartbeater)
		subject.PostMetricEvents([]*messages.MetricEvent{
			{
				Labels:  map[string]string{"d1": "a"},
				Metrics: []*messages.Metric{{Value: 1, EventTime: time.Unix(1234567891, 0)}},
			},
			{
				Labels:  map[string]string{"d2": "a"},
				Metrics: []*messages.Metric{{Value: 2, EventTime: time.Unix(1234567892, 0)}},
			},
			{
				Labels:  map[string]string{"d3": "a"},
				Metrics: []*messages.Metric{{Value: 3, EventTime: time.Unix(1234567893, 0)}},
			},
			{
				Labels:  map[string]string{"d3": "a"},
				Metrics: []*messages.Metric{{Value: 4, EventTime: time.Unix(1234567895, 0)}},
			},
			{
				Labels:  map[string]string{"d3": "a"},
				Metrics: []*messages.Metric{{Value: 5, EventTime: time.Unix(1234567894, 0)}},
			},
		})

		Eventually(metricAdapter.GetPostedMetricEvents).Should(HaveLen(3))
		expected := sortableMetrics{
			{
				Labels:  map[string]string{"d1": "a"},
				Metrics: []*messages.Metric{{Value: 1, EventTime: time.Unix(1234567891, 0)}},
			},
			{
				Labels:  map[string]string{"d2": "a"},
				Metrics: []*messages.Metric{{Value: 2, EventTime: time.Unix(1234567892, 0)}},
			},
			{
				Labels:  map[string]string{"d3": "a"},
				Metrics: []*messages.Metric{{Value: 4, EventTime: time.Unix(1234567895, 0)}},
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
		subject, _ := NewAutoCulledMetricsBuffer(context.TODO(), logger, d, metricAdapter, heartbeater)

		subject.PostMetricEvents([]*messages.MetricEvent{
			{
				Labels:  map[string]string{"Name": "a"},
				Metrics: []*messages.Metric{{Value: 1, EventTime: time.Unix(1234567891, 0)}},
			},
			{
				Labels:  map[string]string{"Name": "b"},
				Metrics: []*messages.Metric{{Value: 2, EventTime: time.Unix(1234567891, 0)}},
			},
		})
		Expect(metricAdapter.GetPostedMetricEvents()).Should(HaveLen(0))
		Eventually(metricAdapter.GetPostedMetricEvents).Should(HaveLen(2))
	})

	It("it flushes metrics when the context is canceled", func() {
		d := 500 * time.Second
		ctx, cancel := context.WithCancel(context.Background())
		subject, _ := NewAutoCulledMetricsBuffer(ctx, logger, d, metricAdapter, heartbeater)

		subject.PostMetricEvents([]*messages.MetricEvent{
			{
				Labels:  map[string]string{"Name": "a"},
				Metrics: []*messages.Metric{{Value: 1, EventTime: time.Unix(1234567891, 0)}},
			},
			{
				Labels:  map[string]string{"Name": "b"},
				Metrics: []*messages.Metric{{Value: 2, EventTime: time.Unix(1234567891, 0)}},
			},
		})
		cancel()
		Eventually(metricAdapter.GetPostedMetricEvents).Should(HaveLen(2))
	})

	It("sends errors through the error channel", func() {
		d := 1 * time.Millisecond
		subject, errs := NewAutoCulledMetricsBuffer(context.TODO(), logger, d, metricAdapter, heartbeater)

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

			subject, _ = NewAutoCulledMetricsBuffer(context.TODO(), logger, 1*time.Millisecond, metricAdapter, heartbeater)
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
