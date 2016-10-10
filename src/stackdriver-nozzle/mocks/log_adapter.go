package mocks

import "github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/stackdriver"

type LogAdapter struct {
	PostedLogs []stackdriver.Log
}

func (m *LogAdapter) PostLog(log *stackdriver.Log) {
	m.PostedLogs = append(m.PostedLogs, *log)
}
