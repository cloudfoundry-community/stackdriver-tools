/*
 * Copyright 2017 Google Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *    http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package mocks

import (
	"sync"

	"code.cloudfoundry.org/lager"
)

type MockLogger struct {
	logs  []Log
	mutex sync.Mutex
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
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.logs = append(m.logs, Log{
		Level:  lager.INFO,
		Action: action,
		Datas:  data,
	})

}

func (m *MockLogger) Error(action string, err error, data ...lager.Data) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.logs = append(m.logs, Log{
		Level:  lager.ERROR,
		Action: action,
		Err:    err,
		Datas:  data,
	})
}

func (m *MockLogger) Fatal(action string, err error, data ...lager.Data) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.logs = append(m.logs, Log{
		Level:  lager.FATAL,
		Action: action,
		Err:    err,
		Datas:  data,
	})
}

func (m *MockLogger) WithData(lager.Data) lager.Logger {
	panic("NYI")
}

func (m *MockLogger) LastLog() Log {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	if len(m.logs) == 0 {
		return Log{}
	}
	return m.logs[len(m.logs)-1]
}

func (m *MockLogger) Logs() []Log {
	m.mutex.Lock()
	defer m.mutex.Unlock()

	return m.logs
}
