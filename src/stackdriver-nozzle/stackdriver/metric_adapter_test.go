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

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/mocks"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/stackdriver"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	labelpb "google.golang.org/genproto/googleapis/api/label"
	metricpb "google.golang.org/genproto/googleapis/api/metric"
	monitoringpb "google.golang.org/genproto/googleapis/monitoring/v3"
)

var _ = Describe("MetricAdapter", func() {
	var (
		subject     stackdriver.MetricAdapter
		client      *mockClient
		heartbeater *mocks.Heartbeater
	)

	BeforeEach(func() {
		client = &mockClient{}
		heartbeater = mocks.NewHeartbeater()
		subject, _ = stackdriver.NewMetricAdapter("my-awesome-project", client, heartbeater)
	})

	It("takes metrics and posts a time series", func() {
		eventTime := time.Now()

		metrics := []stackdriver.Metric{
			{
				Name:  "metricName",
				Value: 123.45,
				Labels: map[string]string{
					"key": "name",
				},
				EventTime: eventTime,
			},
			{
				Name:  "secondMetricName",
				Value: 54.321,
				Labels: map[string]string{
					"secondKey": "secondName",
				},
				EventTime: eventTime,
			},
		}

		subject.PostMetrics(metrics)

		Expect(client.metricReqs).To(HaveLen(1))

		req := client.metricReqs[0]
		Expect(req.Name).To(Equal("projects/my-awesome-project"))

		timeSerieses := req.GetTimeSeries()
		Expect(timeSerieses).To(HaveLen(len(metrics)))

		timeSeries := timeSerieses[0]
		Expect(timeSeries.GetMetric().Type).To(Equal("custom.googleapis.com/metricName"))
		Expect(timeSeries.GetMetric().Labels).To(Equal(metrics[0].Labels))
		Expect(timeSeries.GetPoints()).To(HaveLen(1))

		point := timeSeries.GetPoints()[0]
		Expect(point.GetInterval().GetEndTime().Seconds).To(Equal(int64(eventTime.Unix())))
		Expect(point.GetInterval().GetEndTime().Nanos).To(Equal(int32(eventTime.Nanosecond())))
		value, ok := point.GetValue().GetValue().(*monitoringpb.TypedValue_DoubleValue)
		Expect(ok).To(BeTrue())
		Expect(value.DoubleValue).To(Equal(123.45))

		timeSeries = timeSerieses[1]
		Expect(timeSeries.GetMetric().Type).To(Equal("custom.googleapis.com/secondMetricName"))
		Expect(timeSeries.GetMetric().Labels).To(Equal(metrics[1].Labels))
		Expect(timeSeries.GetPoints()).To(HaveLen(1))

		point = timeSeries.GetPoints()[0]
		value, ok = point.GetValue().GetValue().(*monitoringpb.TypedValue_DoubleValue)
		Expect(ok).To(BeTrue())
		Expect(value.DoubleValue).To(Equal(54.321))
	})

	It("creates metric descriptors", func() {
		metrics := []stackdriver.Metric{
			{
				Name:   "metricWithUnit",
				Labels: map[string]string{"key": "value"},
				Unit:   "{foobar}",
			},
			{
				Name:   "metricWithoutUnit",
				Labels: map[string]string{"key": "value"},
			},
		}

		subject.PostMetrics(metrics)

		Expect(client.descriptorReqs).To(HaveLen(1))
		req := client.descriptorReqs[0]
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
		metrics := []stackdriver.Metric{
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

		Expect(client.descriptorReqs).To(HaveLen(2))
	})

	It("handles concurrent metric descriptor creation", func() {
		metricsWithName := func(name string) []stackdriver.Metric {
			return []stackdriver.Metric{
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

		go subject.PostMetrics(metricsWithName("a"))
		go subject.PostMetrics(metricsWithName("b"))
		go subject.PostMetrics(metricsWithName("a"))
		go subject.PostMetrics(metricsWithName("c"))
		go subject.PostMetrics(metricsWithName("b"))

		Eventually(func() int {
			mutex.Lock()
			defer mutex.Unlock()

			return callCount
		}).Should(Equal(3))
	})

	It("returns the adapter even if we fail to list the metric descriptors", func() {
		expectedErr := errors.New("fail")
		client.listErr = expectedErr
		subject, err := stackdriver.NewMetricAdapter("my-awesome-project", client, heartbeater)
		Expect(subject).To(Not(BeNil()))
		Expect(err).To(Equal(expectedErr))
	})

	It("increments metrics counters", func() {
		metrics := []stackdriver.Metric{
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
		Expect(heartbeater.GetCount("metrics.count")).To(Equal(3))
		Expect(heartbeater.GetCount("metrics.requests")).To(Equal(1))

		subject.PostMetrics(metrics)
		Expect(heartbeater.GetCount("metrics.count")).To(Equal(6))
		Expect(heartbeater.GetCount("metrics.requests")).To(Equal(2))
	})
})

type mockClient struct {
	metricReqs     []*monitoringpb.CreateTimeSeriesRequest
	descriptorReqs []*monitoringpb.CreateMetricDescriptorRequest
	listErr        error
	mutex          sync.Mutex

	CreateMetricDescriptorFn func(request *monitoringpb.CreateMetricDescriptorRequest) error
}

func (mc *mockClient) Post(req *monitoringpb.CreateTimeSeriesRequest) error {
	mc.mutex.Lock()
	mc.metricReqs = append(mc.metricReqs, req)
	mc.mutex.Unlock()

	return nil
}

func (mc *mockClient) CreateMetricDescriptor(request *monitoringpb.CreateMetricDescriptorRequest) error {
	if mc.CreateMetricDescriptorFn != nil {
		return mc.CreateMetricDescriptorFn(request)
	}

	mc.mutex.Lock()
	mc.descriptorReqs = append(mc.descriptorReqs, request)
	mc.mutex.Unlock()

	return nil
}

func (mc *mockClient) ListMetricDescriptors(request *monitoringpb.ListMetricDescriptorsRequest) ([]*metricpb.MetricDescriptor, error) {
	if mc.listErr != nil {
		return nil, mc.listErr
	}
	return []*metricpb.MetricDescriptor{
		{Name: "anExistingMetric"},
	}, nil
}
