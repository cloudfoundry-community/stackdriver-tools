package heartbeat

import (
	"time"

	"errors"

	"github.com/cloudfoundry/lager"
)

type Heartbeater interface {
	Start()
	AddCounter()
	Stop()
}

type heartbeat struct {
	logger  lager.Logger
	trigger <-chan time.Time
	counter chan struct{}
	done    chan struct{}
	started bool
}

func NewHeartbeat(logger lager.Logger, trigger <-chan time.Time) Heartbeater {
	counter := make(chan struct{})
	done := make(chan struct{})
	return &heartbeat{
		logger:  logger,
		trigger: trigger,
		counter: counter,
		done:    done,
		started: false,
	}
}

func (h *heartbeat) Start() {
	h.started = true
	go func() {
		eventCount := 0
		for {
			select {
			case <-h.trigger:
				h.logger.Info("counter", lager.Data{
					"eventCount": eventCount,
				})
				eventCount = 0
			case <-h.counter:
				eventCount++
			case <-h.done:
				h.logger.Info("counterStopped", lager.Data{
					"remainingCount": eventCount,
				})
				return
			}
		}
	}()
}

func (h *heartbeat) AddCounter() {
	if h.started {
		h.counter <- struct{}{}
	} else {
		h.logger.Error("addCounter", errors.New("attempted to add to counter without starting heartbeat"))
	}
}

func (h *heartbeat) Stop() {
	h.done <- struct{}{}
	h.started = false
}
