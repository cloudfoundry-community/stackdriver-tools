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

package metricspipeline

import (
	"context"
	"sort"
	"time"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/messages"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/mocks"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("autoCulledMetricsBuffer", func() {
	var (
		metricAdapter *mocks.MetricAdapter
		logger        *mocks.MockLogger
	)

	BeforeEach(func() {
		metricAdapter = &mocks.MetricAdapter{}
		logger = &mocks.MockLogger{}
		eventsSampledCount.Set(0)
	})

	It("culls duplicate metrics", func() {
		subject := NewAutoCulledMetricsBuffer(context.Background(), logger, 100*time.Millisecond, metricAdapter)

		subject.PostMetrics([]*messages.Metric{
			{
				Name:      "a",
				Labels:    map[string]string{"Name": "a"},
				Value:     1,
				EventTime: time.Unix(1234567890, 0),
			},
			{
				Name:      "b",
				Labels:    map[string]string{"Name": "a"},
				Value:     0,
				EventTime: time.Unix(1234567890, 0),
			},
			{
				Name:      "a",
				Labels:    map[string]string{"Name": "a"},
				Value:     2,
				EventTime: time.Unix(1234567891, 0),
			},
			{
				Name:      "b",
				Labels:    map[string]string{"Name": "a"},
				Value:     2,
				EventTime: time.Unix(1234567891, 0),
			},
		})
		Eventually(metricAdapter.GetPostedMetrics).Should(HaveLen(2))

		expected := sortableMetrics{
			{
				Name:      "a",
				Labels:    map[string]string{"Name": "a"},
				Value:     2,
				EventTime: time.Unix(1234567891, 0),
			},
			{
				Name:      "b",
				Labels:    map[string]string{"Name": "a"},
				Value:     2,
				EventTime: time.Unix(1234567891, 0),
			},
		}
		actual := sortableMetrics(metricAdapter.GetPostedMetrics())
		sort.Sort(expected)
		sort.Sort(actual)

		Expect(actual).To(BeEquivalentTo(expected))
		Expect(eventsSampledCount.IntValue()).To(Equal(2))
	})

	It("culls multiple duplicates, keeping the latest", func() {
		subject := NewAutoCulledMetricsBuffer(context.Background(), logger, 100*time.Millisecond, metricAdapter)
		subject.PostMetrics([]*messages.Metric{
			{
				Labels:    map[string]string{"d1": "a"},
				Value:     1,
				EventTime: time.Unix(1234567891, 0),
			},
			{
				Labels:    map[string]string{"d2": "a"},
				Value:     2,
				EventTime: time.Unix(1234567892, 0),
			},
			{
				Labels:    map[string]string{"d3": "a"},
				Value:     3,
				EventTime: time.Unix(1234567893, 0),
			},
			{
				Labels:    map[string]string{"d3": "a"},
				Value:     4,
				EventTime: time.Unix(1234567895, 0),
			},
			{
				Labels:    map[string]string{"d3": "a"},
				Value:     5,
				EventTime: time.Unix(1234567894, 0),
			},
		})

		Eventually(metricAdapter.GetPostedMetrics).Should(HaveLen(3))
		expected := sortableMetrics{
			{
				Labels:    map[string]string{"d1": "a"},
				Value:     1,
				EventTime: time.Unix(1234567891, 0),
			},
			{
				Labels:    map[string]string{"d2": "a"},
				Value:     2,
				EventTime: time.Unix(1234567892, 0),
			},
			{
				Labels:    map[string]string{"d3": "a"},
				Value:     4,
				EventTime: time.Unix(1234567895, 0),
			},
		}
		actual := sortableMetrics(metricAdapter.GetPostedMetrics())

		sort.Sort(expected)
		sort.Sort(actual)

		Expect(actual).To(BeEquivalentTo(expected))
		Expect(eventsSampledCount.IntValue()).To(Equal(2))
	})

	It("it buffers metrics for the expected duration before flushing", func() {
		d := 500 * time.Millisecond
		subject := NewAutoCulledMetricsBuffer(context.Background(), logger, d, metricAdapter)

		subject.PostMetrics([]*messages.Metric{
			{
				Labels:    map[string]string{"Name": "a"},
				Value:     1,
				EventTime: time.Unix(1234567891, 0),
			},
			{
				Labels:    map[string]string{"Name": "b"},
				Value:     2,
				EventTime: time.Unix(1234567891, 0),
			},
		})
		Expect(metricAdapter.GetPostedMetrics()).Should(HaveLen(0))
		Eventually(metricAdapter.GetPostedMetrics).Should(HaveLen(2))
	})

	It("it flushes metrics when the context is canceled", func() {
		d := 500 * time.Second
		ctx, cancel := context.WithCancel(context.Background())
		subject := NewAutoCulledMetricsBuffer(ctx, logger, d, metricAdapter)

		subject.PostMetrics([]*messages.Metric{
			{
				Labels:    map[string]string{"Name": "a"},
				Value:     1,
				EventTime: time.Unix(1234567891, 0),
			},
			{
				Labels:    map[string]string{"Name": "b"},
				Value:     2,
				EventTime: time.Unix(1234567891, 0),
			},
		})
		cancel()
		Eventually(metricAdapter.GetPostedMetrics).Should(HaveLen(2))
	})

	Describe("with a slow MetricAdapter", func() {
		var (
			metricPosted chan interface{}
			subject      MetricsBuffer
		)

		BeforeEach(func() {
			metricPosted = make(chan interface{})
			metricAdapter.PostMetricsFn = func([]*messages.Metric) error {
				metricPosted <- struct{}{}
				time.Sleep(30 * time.Second)
				return nil
			}

			subject = NewAutoCulledMetricsBuffer(context.Background(), logger, 1*time.Millisecond, metricAdapter)
		})

		It("doesn't block new metrics during flush", func() {
			metric := []*messages.Metric{{}}
			subject.PostMetrics(metric)

			Eventually(metricPosted).Should(Receive())
			unblocked := make(chan interface{})
			go func() {
				subject.PostMetrics(metric)
				unblocked <- struct{}{}
			}()
			Eventually(unblocked).Should(Receive())
		})
	})
})

type sortableMetrics []*messages.Metric

func (b sortableMetrics) Len() int {
	return len(b)
}
func (b sortableMetrics) Swap(i, j int) {
	b[i], b[j] = b[j], b[i]
}
func (b sortableMetrics) Less(i, j int) bool {
	return b[i].Hash() < b[j].Hash()
}
