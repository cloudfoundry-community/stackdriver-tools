package metrics_router_test

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"

	"testing"
)

func TestMetricsRouter(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "MetricsRouter Suite")
}
