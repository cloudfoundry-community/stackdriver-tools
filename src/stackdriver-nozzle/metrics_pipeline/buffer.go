package metrics_pipeline

import "github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/stackdriver"

type MetricsBuffer interface {
	stackdriver.MetricAdapter
	IsEmpty() bool
}
