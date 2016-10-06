package heartbeat_test

import (
	"github.com/cloudfoundry/lager"

	"time"

	"stackdriver-nozzle/heartbeat"

	"errors"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

var _ = Describe("Heartbeat", func() {
	var (
		subject heartbeat.Heartbeater
		logger  *mockLogger
		trigger chan time.Time
	)

	BeforeEach(func() {
		logger = &mockLogger{}
		trigger = make(chan time.Time)

		subject = heartbeat.NewHeartbeat(logger, trigger)
		subject.Start()
	})

	It("should start at zero", func() {
		trigger <- time.Now()

		Eventually(func() log {
			return logger.lastLog()
		}).Should(Equal(log{
			level:  lager.INFO,
			action: "counter",
			datas: []lager.Data{
				{"eventCount": 0},
			},
		}))
	})

	It("should count events", func() {
		for i := 0; i < 10; i++ {
			subject.AddCounter()
		}

		trigger <- time.Now()

		Eventually(func() log {
			return logger.lastLog()
		}).Should(Equal(log{
			level:  lager.INFO,
			action: "counter",
			datas: []lager.Data{
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

		Eventually(func() log {
			return logger.lastLog()
		}).Should(Equal(log{
			level:  lager.INFO,
			action: "counter",
			datas: []lager.Data{
				{"eventCount": 5},
			},
		}))
	})

	It("should stop counting", func() {
		for i := 0; i < 5; i++ {
			subject.AddCounter()
		}
		subject.Stop()

		Eventually(func() log {
			return logger.lastLog()
		}).Should(Equal(log{
			level:  lager.INFO,
			action: "counterStopped",
			datas: []lager.Data{
				{"remainingCount": 5},
			},
		}))

		subject.AddCounter()
		Expect(logger.lastLog()).To(Equal(log{
			level:  lager.ERROR,
			action: "addCounter",
			err:    errors.New("attempted to add to counter without starting heartbeat"),
		}))
	})
})

type mockLogger struct {
	logs []log
}

type log struct {
	level  lager.LogLevel
	action string
	err    error
	datas  []lager.Data
}

func (m *mockLogger) RegisterSink(lager.Sink) {
	panic("NYI")
}

func (m *mockLogger) Session(task string, data ...lager.Data) lager.Logger {
	panic("NYI")
}

func (m *mockLogger) SessionName() string {
	panic("NYI")
}

func (m *mockLogger) Debug(action string, data ...lager.Data) {
	panic("NYI")
}

func (m *mockLogger) Info(action string, data ...lager.Data) {
	m.logs = append(m.logs, log{
		level:  lager.INFO,
		action: action,
		datas:  data,
	})
}

func (m *mockLogger) Error(action string, err error, data ...lager.Data) {
	m.logs = append(m.logs, log{
		level:  lager.ERROR,
		action: action,
		err:    err,
		datas:  data,
	})
}

func (m *mockLogger) Fatal(action string, err error, data ...lager.Data) {
	panic("NYI")
}

func (m *mockLogger) WithData(lager.Data) lager.Logger {
	panic("NYI")
}

func (m *mockLogger) lastLog() log {
	if len(m.logs) == 0 {
		return log{}
	}
	return m.logs[len(m.logs)-1]
}
