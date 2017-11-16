package cloudfoundry

import (
	"fmt"
	"io"
)

type StdoutWriter struct {
	writer io.Writer
}

func (w *StdoutWriter) Emit(needle string, count int) error {
	for i := 0; i < count; i++ {
		fmt.Fprintf(w.writer, needle)
	}

	return nil
}

func NewLogWriter(writer io.Writer) *StdoutWriter {
	return &StdoutWriter{writer}
}
