package session

import (
	"context"
	"crypto/rand"
	"fmt"
	"io"
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

func (s Session) Run(ctx context.Context) Result {
	const count = 10
	needle := getNeedle()

	s.emitter.Emit(needle, count)

	doneChan := ctx.Done()
	if doneChan == nil {
		panic("context can not be cancelled")
	}
	<-ctx.Done()

	found, _ := s.probe.Find(needle, count)
	return Result{float64(count-found) / count}
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
