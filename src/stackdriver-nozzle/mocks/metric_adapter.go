package mocks

import (
	"sync"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/stackdriver"
)

type MetricAdapter struct {
	PostMetricsFn   func(metrics []stackdriver.Metric) error
	PostMetricError error
	postedMetrics   []stackdriver.Metric
	mutex           sync.Mutex
}

func (m *MetricAdapter) PostMetrics(metrics []stackdriver.Metric) error {
	if m.PostMetricsFn != nil {
		return m.PostMetricsFn(metrics)
	}

	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.postedMetrics = append(m.postedMetrics, metrics...)
	return m.PostMetricError
}

func (m *MetricAdapter) GetPostedMetrics() []stackdriver.Metric {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.postedMetrics
}
