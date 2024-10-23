package concurrency

import (
	"context"
	"sync"
)

type Limiter struct {
	limit   chan struct{}
	working sync.WaitGroup
}

func NewLimiter(n int) *Limiter {
	return &Limiter{limit: make(chan struct{}, n)}
}

func (lim *Limiter) Go(ctx context.Context, fn func()) bool {
	// ensure that we aren't trying to start when the
	// context has been cancelled.
	if ctx.Err() != nil {
		return false
	}

	// wait until we can start a goroutine:
	select {
	case lim.limit <- struct{}{}:
	case <-ctx.Done():
		// maybe the user got tired of waiting?
		return false
	}

	lim.working.Add(1)
	go func() {
		defer func() {
			<-lim.limit
			lim.working.Done()
		}()

		fn()
	}()

	return true
}

func (lim *Limiter) Wait() {
	lim.working.Wait()
}
