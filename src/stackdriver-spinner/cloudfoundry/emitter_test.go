package cloudfoundry_test

import (
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-spinner/cloudfoundry"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-spinner/fakes"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Emitter", func() {
	It("logs to stdout once", func() {
		mockWriter := fakes.Writer{}

		writer := cloudfoundry.NewEmitter(&mockWriter)
		writer.Emit("something", 1, 0)

		Expect(mockWriter.Writes).To(HaveLen(1))
		Expect(mockWriter.Writes[0]).To(ContainSubstring("something"))
	})

	It("logs to stdout x specified times", func() {
		mockWriter := fakes.Writer{}

		writer := cloudfoundry.NewEmitter(&mockWriter)
		writer.Emit("something", 10, 0)

		Expect(mockWriter.Writes).To(HaveLen(10))
	})
})
