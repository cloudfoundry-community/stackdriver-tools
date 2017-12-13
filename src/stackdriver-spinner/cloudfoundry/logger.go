package cloudfoundry

import (
	"fmt"
	"io"
)

type Logger struct {
	writer io.Writer
}

func (w *Logger) Emit(message string, count int) error {
	for i := 0; i < count; i++ {
		_, err := fmt.Fprintf(w.writer, message+" count: %d \n", i)
		if err != nil {
			return err
		}
	}
	return nil
}

func NewEmitter(writer io.Writer) *Logger {
	return &Logger{writer}
}
