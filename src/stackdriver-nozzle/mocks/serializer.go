package mocks

import (
	"github.com/cloudfoundry/sonde-go/events"
	"stackdriver-nozzle/serializer"
)

type MockSerializer struct {
	GetLogFn     func(*events.Envelope) *serializer.Log
	GetMetricsFn func(*events.Envelope) ([]*serializer.Metric, error)
	IsLogFn      func(*events.Envelope) bool
}

func (m *MockSerializer) GetLog(envelope *events.Envelope) *serializer.Log {
	if m.GetLogFn != nil {
		return m.GetLogFn(envelope)
	}
	return nil
}

func (m *MockSerializer) GetMetrics(envelope *events.Envelope) ([]*serializer.Metric, error) {
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
