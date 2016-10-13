package mocks

import "github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/stackdriver"

type MetricsBuffer struct {
	PostedMetrics []stackdriver.Metric
}

func (m *MetricsBuffer) PostMetric(metric *stackdriver.Metric) {
	m.PostedMetrics = append(m.PostedMetrics, *metric)
}
