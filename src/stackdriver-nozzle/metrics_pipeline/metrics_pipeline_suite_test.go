package metrics_pipeline_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestMetricsBuffer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MetricsPipeline Suite")
}
