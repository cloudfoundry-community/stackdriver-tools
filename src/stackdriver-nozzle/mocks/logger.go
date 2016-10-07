package mocks

import "github.com/cloudfoundry/lager"

type MockLogger struct {
	logs []Log
}

type Log struct {
	Level  lager.LogLevel
	Action string
	Err    error
	Datas  []lager.Data
}

func (m *MockLogger) RegisterSink(lager.Sink) {
	panic("NYI")
}

func (m *MockLogger) Session(task string, data ...lager.Data) lager.Logger {
	panic("NYI")
}

func (m *MockLogger) SessionName() string {
	panic("NYI")
}

func (m *MockLogger) Debug(action string, data ...lager.Data) {
	panic("NYI")
}

func (m *MockLogger) Info(action string, data ...lager.Data) {
	m.logs = append(m.logs, Log{
		Level:  lager.INFO,
		Action: action,
		Datas:  data,
	})
}

func (m *MockLogger) Error(action string, err error, data ...lager.Data) {
	m.logs = append(m.logs, Log{
		Level:  lager.ERROR,
		Action: action,
		Err:    err,
		Datas:  data,
	})
}

func (m *MockLogger) Fatal(action string, err error, data ...lager.Data) {
	panic("NYI")
}

func (m *MockLogger) WithData(lager.Data) lager.Logger {
	panic("NYI")
}

func (m *MockLogger) LastLog() Log {
	if len(m.logs) == 0 {
		return Log{}
	}
	return m.logs[len(m.logs)-1]
}
