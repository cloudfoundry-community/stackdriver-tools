package fakes

type MockReaderWriter struct {
	MockWriter

	FindFn func(string, int) (int, error)
}

func (m *MockReaderWriter) Find(needle string, count int) (int, error) {
	if m.FindFn != nil {
		return m.FindFn(needle, count)
	}

	numMatches := 0
	for _, content := range m.Writes {
		if content == needle {
			numMatches++
		}
	}

	return numMatches, nil
}
