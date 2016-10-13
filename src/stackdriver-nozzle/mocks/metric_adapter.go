package mocks

import "github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/stackdriver"

type MetricAdapter struct {
	PostMetricsFn   func(metrics []stackdriver.Metric) error
	PostedMetrics   []stackdriver.Metric
	PostMetricError error
}

func (m *MetricAdapter) PostMetrics(metrics []stackdriver.Metric) error {
	if m.PostMetricsFn != nil {
		return m.PostMetricsFn(metrics)
	}
	m.PostedMetrics = append(m.PostedMetrics, metrics...)
	return m.PostMetricError
}
