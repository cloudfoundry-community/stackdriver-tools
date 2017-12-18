package cloudfoundry

import (
	"fmt"
	"io"
	"time"
)

type Emitter struct {
	writer io.Writer
}

func (w *Emitter) Emit(message string, count int, wait time.Duration) error {
	for i := 0; i < count; i++ {
		_, err := fmt.Fprintf(w.writer, message+" count: %d \n", i)
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
