package fakes

type LosslessProbe struct {
}

func (m *LosslessProbe) Find(needle string, count int) (int, error) {
	return count, nil
}

type ConfigurableProbe struct {
	FindFunc func(string, int) (int, error)
}

func (m *ConfigurableProbe) Find(needle string, count int) (int, error) {
	return m.FindFunc(needle, count)
}
