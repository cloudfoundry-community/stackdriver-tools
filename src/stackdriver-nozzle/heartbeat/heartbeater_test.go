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

var _ = Describe("Heartbeater", func() {
	var (
		subject heartbeat.Heartbeater
		logger  *mocks.MockLogger
		trigger chan time.Time
	)

	BeforeEach(func() {
		logger = &mocks.MockLogger{}
		trigger = make(chan time.Time)

		subject = heartbeat.NewHeartbeater(logger, trigger)
		subject.Start()
	})

	It("should start at zero", func() {
		trigger <- time.Now()

		Eventually(func() mocks.Log {
			return logger.LastLog()
		}).Should(Equal(mocks.Log{
			Level:  lager.INFO,
			Action: "heartbeater",
			Datas: []lager.Data{
				{"counters": map[string]uint{}},
			},
		}))
	})

	It("should count events", func() {
		for i := 0; i < 10; i++ {
			subject.Increment("foo")
		}

		trigger <- time.Now()

		Eventually(func() mocks.Log {
			return logger.LastLog()
		}).Should(Equal(mocks.Log{
			Level:  lager.INFO,
			Action: "heartbeater",
			Datas: []lager.Data{
				{"counters": map[string]uint{"foo": 10}},
			},
		}))
	})

	It("should reset the heartbeater on triggers", func() {
		for i := 0; i < 10; i++ {
			subject.Increment("foo")
		}

		trigger <- time.Now()

		for i := 0; i < 5; i++ {
			subject.Increment("foo")
		}

		trigger <- time.Now()

		Eventually(func() mocks.Log {
			return logger.LastLog()
		}).Should(Equal(mocks.Log{
			Level:  lager.INFO,
			Action: "heartbeater",
			Datas: []lager.Data{
				{"counters": map[string]uint{"foo": 5}},
			},
		}))
	})

	It("should stop counting", func() {
		for i := 0; i < 5; i++ {
			subject.Increment("foo")
		}
		subject.Stop()

		Eventually(func() mocks.Log {
			return logger.LastLog()
		}).Should(Equal(mocks.Log{
			Level:  lager.INFO,
			Action: "heartbeater",
			Datas: []lager.Data{
				{"counters": map[string]uint{"foo": 5}},
			},
		}))

		subject.Increment("foo")
		Expect(logger.LastLog()).To(Equal(mocks.Log{
			Level:  lager.ERROR,
			Action: "heartbeater",
			Err:    errors.New("attempted to increment counter without starting heartbeater"),
		}))
	})

	It("can count multiple events", func() {
		for i := 0; i < 10; i++ {
			subject.Increment("foo")
		}

		for i := 0; i < 5; i++ {
			subject.Increment("bar")
		}

		trigger <- time.Now()

		Eventually(func() mocks.Log {
			return logger.LastLog()
		}).Should(Equal(mocks.Log{
			Level:  lager.INFO,
			Action: "heartbeater",
			Datas: []lager.Data{
				{"counters": map[string]uint{
					"foo": 10,
					"bar": 5,
				}},
			},
		}))
	})
})
