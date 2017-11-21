package session

import (
	"crypto/rand"
	"fmt"
	"io"
	"time"
)

type Emitter interface {
	Emit(needle string, count int) error
}

type Probe interface {
	Find(needle string, count int) (int, error)
}

type Session struct {
	emitter Emitter
	probe   Probe
}

type Result struct {
	Loss float64
}

func NewSession(emitter Emitter, probe Probe) Session {
	return Session{emitter, probe}
}

func (s Session) Run(count int, waitTime time.Duration) (Result, error) {
	needle := getNeedle()
	err := s.emitter.Emit(needle, count)
	if err != nil {
		return Result{}, err
	}

	time.Sleep(waitTime)

	found, err := s.probe.Find(needle, count)
	if err != nil {
		return Result{}, err
	}
	return Result{float64(count-found) / float64(count)}, nil
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
