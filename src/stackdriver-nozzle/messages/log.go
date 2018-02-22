package messages

import "cloud.google.com/go/logging"

type Log struct {
	Payload  interface{}
	Labels   map[string]string
	Severity logging.Severity
}
