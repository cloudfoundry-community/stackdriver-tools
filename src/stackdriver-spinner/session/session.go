package session

import (
	"crypto/rand"
	"fmt"
	"io"
	"time"
)

type Emitter interface {
	Emit(needle string) (int, error)
}

type Probe interface {
	Find(start time.Time, needle string, count int) (int, error)
}

type Session struct {
	emitter Emitter
	probe   Probe
}

type Result struct {
	GUID  string
	Found int
	Loss  float64
}

func NewSession(emitter Emitter, probe Probe) Session {
	return Session{emitter, probe}
}

func (s Session) Run(burstInterval time.Duration) (Result, error) {
	needle := getNeedle()
	emitted, err := s.emitter.Emit(needle)
	if err != nil {
		return Result{}, err
	}

	queryTime := time.Now().Add(-burstInterval - 10)
	time.Sleep(burstInterval)

	found, err := s.probe.Find(queryTime, needle, emitted)
	if err != nil {
		return Result{}, err
	}
	return Result{needle, found, float64(emitted-found) / float64(emitted)}, nil
}

func getNeedle() string {
	uuid := make([]byte, 16)
	n, err := io.ReadFull(rand.Reader, uuid)
	if n != len(uuid) || err != nil {
		panic("failed to generate a needle")
	}
	needle := fmt.Sprintf("%x", uuid)
	return needle
}
