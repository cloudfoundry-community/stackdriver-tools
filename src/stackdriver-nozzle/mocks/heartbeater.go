package mocks

import "sync"

func NewHeartbeater() *Heartbeater {
	return &Heartbeater{counters: map[string]int{}}
}

type Heartbeater struct {
	started  bool
	counters map[string]int
	mutex    sync.Mutex
}

func (h *Heartbeater) Start() {
	h.mutex.Lock()
	h.started = true
	h.mutex.Unlock()
}

func (h *Heartbeater) Increment(name string) {
	h.mutex.Lock()
	h.counters[name] += 1
	h.mutex.Unlock()
}

func (h *Heartbeater) Stop() {
	h.mutex.Lock()
	h.started = false
	h.mutex.Unlock()
}

func (h *Heartbeater) IsRunning() bool {
	h.mutex.Lock()
	defer h.mutex.Unlock()
	return h.started
}

func (h *Heartbeater) GetCount(name string) int {
	h.mutex.Lock()
	defer h.mutex.Unlock()

	return h.counters[name]
}
