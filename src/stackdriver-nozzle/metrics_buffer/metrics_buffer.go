package metrics_buffer

import "github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/stackdriver"

type MetricsBuffer interface {
	stackdriver.MetricAdapter
	IsEmpty() bool
}
