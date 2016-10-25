package mocks

import "sync"

func New() *Heartbeater {
	return &Heartbeater{Counters: map[string]int{}}
}

type Heartbeater struct {
	Started  bool
	Counters map[string]int
	mutex    sync.Mutex
}

func (h *Heartbeater) Start() {
	h.Started = true
}

func (h *Heartbeater) Increment(name string) {
	h.mutex.Lock()
	h.Counters[name] += 1
	h.mutex.Unlock()
}

func (h *Heartbeater) Stop() {
	h.Started = false
}
