package fakes

type MockWriter struct {
	Writes []string
}

func (m *MockWriter) Write(p []byte) (n int, err error) {
	m.Writes = append(m.Writes, string(p))

	return len(p), nil
}
