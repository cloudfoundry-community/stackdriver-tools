package fakes

import "time"

type LosslessProbe struct {
}

func (m *LosslessProbe) Find(start time.Time, needle string, count int) (int, error) {
	return count, nil
}

type ConfigurableProbe struct {
	FindFunc func(time.Time, string, int) (int, error)
}

func (m *ConfigurableProbe) Find(start time.Time, needle string, count int) (int, error) {
	return m.FindFunc(start, needle, count)
}
