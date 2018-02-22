package fakes

type Writer struct {
	Writes []string
}

func (m *Writer) Write(p []byte) (n int, err error) {
	m.Writes = append(m.Writes, string(p))

	return len(p), nil
}

type FailingWriter struct {
	Err error
}

func (f *FailingWriter) Write(p []byte) (n int, err error) {
	return 0, f.Err
}
