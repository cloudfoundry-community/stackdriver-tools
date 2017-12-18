package cloudfoundry

import (
	"fmt"
	"io"
	"time"
	"encoding/json"
)

type Emitter struct {
	writer io.Writer
}

type Payload struct {
	Timestamp string `json:"timestamp"`
	GUID      string `json:"guid"`
	Count     int    `json:"count"`
}

func (w *Emitter) Emit(guid string, count int, wait time.Duration) error {
	for i := 1; i <= count; i++ {
		pl := Payload{
			Timestamp: time.Now().UTC().Format("2006-01-02T15:04:05.000-07:00"),
			GUID:      guid,
			Count:     i,
		}

		msg, err := json.Marshal(pl)
		if err != nil {
			return err
		}
		_, err = fmt.Fprintf(w.writer, string(msg)+"\n")
		time.Sleep(wait)
		if err != nil {
			return err
		}
	}
	return nil
}

func NewEmitter(writer io.Writer) *Emitter {
	return &Emitter{writer}
}
