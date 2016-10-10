package mocks

import (
	"github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/stackdriver"
	"github.com/cloudfoundry/sonde-go/events"
)

type MockSerializer struct {
	GetLogFn     func(*events.Envelope) *stackdriver.Log
	GetMetricsFn func(*events.Envelope) ([]stackdriver.Metric, error)
	IsLogFn      func(*events.Envelope) bool
}

func (m *MockSerializer) GetLog(envelope *events.Envelope) *stackdriver.Log {
	if m.GetLogFn != nil {
		return m.GetLogFn(envelope)
	}
	return nil
}

func (m *MockSerializer) GetMetrics(envelope *events.Envelope) ([]stackdriver.Metric, error) {
	if m.GetMetricsFn != nil {
		return m.GetMetricsFn(envelope)
	}
	return nil, nil
}

func (m *MockSerializer) IsLog(envelope *events.Envelope) bool {
	if m.IsLogFn != nil {
		return m.IsLogFn(envelope)
	}
	return true
}
