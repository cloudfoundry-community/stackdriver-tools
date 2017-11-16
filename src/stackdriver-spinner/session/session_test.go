package session_test

import (
	"context"

	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-spinner/cloudfoundry"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-spinner/fakes"
	"github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-spinner/session"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Session", func() {
	var (
		readerWriter fakes.MockReaderWriter
		emitter      *cloudfoundry.StdoutWriter
		s            session.Session
	)
	BeforeEach(func() {
		readerWriter = fakes.MockReaderWriter{}
		emitter = cloudfoundry.NewLogWriter(&readerWriter)
		s = session.NewSession(emitter, &readerWriter)
	})

	Context("async execution", func() {

		It("sends logs before context is done", func() {
			ctx, cancel := context.WithCancel(context.Background())

			go func() { s.Run(ctx) }()

			Eventually(func() int { return len(readerWriter.Writes) }).Should(BeNumerically(">", 0))
			cancel()
		})

		Describe("with mocked find", func() {
			var findCalled func() bool
			BeforeEach(func() {
				called := false
				readerWriter.FindFn = func(s string, i int) (int, error) {
					called = true
					return i, nil
				}
				findCalled = func() bool { return called }
			})

			It("probes after the context is done", func() {
				ctx, cancel := context.WithCancel(context.Background())

				go func() { s.Run(ctx) }()

				Consistently(findCalled).Should(BeFalse())

				cancel()

				Eventually(findCalled).Should(BeTrue())
			})

		})
	})

	It("panics if the context can never be cancelled", func() {
		ctx := context.Background()
		Expect(func() { s.Run(ctx) }).To(Panic())
	})

	Context("synchronous execution", func() {

		ctx, cancel := context.WithCancel(context.Background())
		cancel()

		It("correctly detects zero loss", func() {
			result := s.Run(ctx)
			Expect(result.Loss).To(Equal(0.0))
		})

		It("measures loss when the emitter drops data", func() {
			readerWriter.FindFn = func(needle string, count int) (int, error) {
				return count - 2, nil
			}

			result := s.Run(ctx)
			Expect(result.Loss).To(BeNumerically(">", 0.0))
		})

		It("does not re-use the same needle", func() {
			needles := map[string]int{}

			for i := 0; i < 10; i++ {
				s.Run(ctx)
				needles[readerWriter.Writes[0]] += 1
				readerWriter.Writes = []string{}
			}

			for _, v := range needles {
				Expect(v).To(Equal(1))
			}
		})

		It(" results", func() {

		})
	})
})
