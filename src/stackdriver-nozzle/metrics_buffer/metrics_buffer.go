package metrics_buffer

import "github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/stackdriver"

type MetricsBuffer interface {
	PostMetrics([]stackdriver.Metric) error
	IsEmpty() bool
}
