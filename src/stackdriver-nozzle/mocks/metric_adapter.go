package mocks

import "github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/stackdriver"

type MetricAdapter struct {
	PostedMetrics   []stackdriver.Metric
	PostMetricError error
}

func (m *MetricAdapter) PostMetrics(metrics []stackdriver.Metric) error {
	m.PostedMetrics = append(m.PostedMetrics, metrics...)
	return m.PostMetricError
}
