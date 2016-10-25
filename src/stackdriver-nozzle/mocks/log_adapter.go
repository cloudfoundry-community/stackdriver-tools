package mocks

import "github.com/cloudfoundry-community/gcp-tools-release/src/stackdriver-nozzle/stackdriver"

type LogAdapter struct {
	PostedLogs []stackdriver.Log
}

func (la *LogAdapter) PostLog(log *stackdriver.Log) {
	la.PostedLogs = append(la.PostedLogs, *log)
}

func (la *LogAdapter) Flush() {
}
