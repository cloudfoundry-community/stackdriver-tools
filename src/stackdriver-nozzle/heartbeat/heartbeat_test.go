package heartbeat_test

import (
	"errors"
	"time"

	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/heartbeat"
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/mocks"
	"github.com/cloudfoundry/lager"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Heartbeat", func() {
	var (
		subject heartbeat.Heartbeater
		logger  *mocks.MockLogger
		trigger chan time.Time
	)

	BeforeEach(func() {
		logger = &mocks.MockLogger{}
		trigger = make(chan time.Time)

		subject = heartbeat.NewHeartbeat(logger, trigger)
		subject.Start()
	})

	It("should start at zero", func() {
		trigger <- time.Now()

		Eventually(func() mocks.Log {
			return logger.LastLog()
		}).Should(Equal(mocks.Log{
			Level:  lager.INFO,
			Action: "counter",
			Datas: []lager.Data{
				{"eventCount": 0},
			},
		}))
	})

	It("should count events", func() {
		for i := 0; i < 10; i++ {
			subject.AddCounter()
		}

		trigger <- time.Now()

		Eventually(func() mocks.Log {
			return logger.LastLog()
		}).Should(Equal(mocks.Log{
			Level:  lager.INFO,
			Action: "counter",
			Datas: []lager.Data{
				{"eventCount": 10},
			},
		}))
	})

	It("should reset the counter on triggers", func() {
		for i := 0; i < 10; i++ {
			subject.AddCounter()
		}

		trigger <- time.Now()

		for i := 0; i < 5; i++ {
			subject.AddCounter()
		}

		trigger <- time.Now()

		Eventually(func() mocks.Log {
			return logger.LastLog()
		}).Should(Equal(mocks.Log{
			Level:  lager.INFO,
			Action: "counter",
			Datas: []lager.Data{
				{"eventCount": 5},
			},
		}))
	})

	It("should stop counting", func() {
		for i := 0; i < 5; i++ {
			subject.AddCounter()
		}
		subject.Stop()

		Eventually(func() mocks.Log {
			return logger.LastLog()
		}).Should(Equal(mocks.Log{
			Level:  lager.INFO,
			Action: "counterStopped",
			Datas: []lager.Data{
				{"remainingCount": 5},
			},
		}))

		subject.AddCounter()
		Expect(logger.LastLog()).To(Equal(mocks.Log{
			Level:  lager.ERROR,
			Action: "addCounter",
			Err:    errors.New("attempted to add to counter without starting heartbeat"),
		}))
	})
})
