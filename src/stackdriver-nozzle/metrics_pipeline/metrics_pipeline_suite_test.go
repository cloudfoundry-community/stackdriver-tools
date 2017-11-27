package metrics_pipeline

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestMetricsBuffer(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MetricsPipeline Suite")
}
