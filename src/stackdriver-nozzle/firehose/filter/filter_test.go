package filter_test

import (
	"github.com/evandbrown/gcp-tools-release/src/stackdriver-nozzle/firehose/filter"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Filter", func() {
	It("can be created", func() {
		f := filter.New()
		Expect(f).NotTo(BeNil())
	})
})
