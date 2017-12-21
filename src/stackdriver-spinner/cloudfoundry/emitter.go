package cloudfoundry

import (
	"encoding/json"
	"fmt"
	"io"
	"time"
)

type Emitter struct {
	writer io.Writer
	count  int
	delay  time.Duration
}

type Payload struct {
	Timestamp string `json:"timestamp"`
	GUID      string `json:"guid"`
	Count     int    `json:"count"`
}

func (e *Emitter) Emit(guid string) (int, error) {
	for i := 0; i < e.count; i++ {
		pl := Payload{
			Timestamp: time.Now().UTC().Format("2006-01-02T15:04:05.000-07:00"),
			GUID:      guid,
			Count:     i + 1,
		}

		msg, err := json.Marshal(pl)
		if err != nil {
			return i, err
		}

		_, err = fmt.Fprintf(e.writer, string(msg)+"\n")
		time.Sleep(e.delay)
		if err != nil {
			return i, err
		}
	}
	return e.count, nil
}

func NewEmitter(writer io.Writer, count int, delay time.Duration) *Emitter {
	return &Emitter{writer, count, delay}
}
