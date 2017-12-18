package session

import (
	"crypto/rand"
	"fmt"
	"io"
	"time"
)

type Emitter interface {
	Emit(needle string, count int, wait time.Duration) error
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

func (s Session) Run(count int, burstInterval time.Duration, sleepTime time.Duration) (Result, error) {
	needle := getNeedle()
	err := s.emitter.Emit(needle, count, sleepTime)
	if err != nil {
		return Result{}, err
	}

	queryTime := time.Now().Add(-burstInterval - 10)
	time.Sleep(burstInterval)

	found, err := s.probe.Find(queryTime, needle, count)
	if err != nil {
		return Result{}, err
	}
	return Result{needle, found, float64(count-found) / float64(count)}, nil
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
