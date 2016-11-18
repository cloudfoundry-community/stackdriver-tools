package mocks

import "github.com/cloudfoundry-community/stackdriver-tools/src/stackdriver-nozzle/stackdriver"

type LogAdapter struct {
	PostedLogs []stackdriver.Log
}

func (la *LogAdapter) PostLog(log *stackdriver.Log) {
	la.PostedLogs = append(la.PostedLogs, *log)
}

func (la *LogAdapter) Flush() {
}
