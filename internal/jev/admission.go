package jev

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Admission caps in-flight calls. Background work stops at total-reserve so
// those slots stay available for interactive filing.
type Admission struct {
	total   int
	reserve int
	mu      sync.Mutex
	cond    *sync.Cond
	used    int
	bgUsed  int
}

func newAdmission(total, reserve int) (*Admission, error) {
	if total < 2 {
		return nil, fmt.Errorf("%w: slot count", ErrRequest)
	}
	if reserve < 1 || reserve >= total {
		return nil, fmt.Errorf("%w: interactive reserve", ErrRequest)
	}
	a := &Admission{total: total, reserve: reserve}
	a.cond = sync.NewCond(&a.mu)
	return a, nil
}

// Acquire waits until ctx ends or a slot this priority may take is free.
// sync.Cond cannot select on ctx, so cancellation broadcasts under the same lock.
func (a *Admission) Acquire(ctx context.Context, p Priority) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()
	go func() {
		<-ctx.Done()
		a.mu.Lock()
		a.cond.Broadcast()
		a.mu.Unlock()
	}()

	for !a.fits(p) {
		if err := ctx.Err(); err != nil {
			return err
		}
		a.cond.Wait()
	}
	if err := ctx.Err(); err != nil {
		return err
	}
	a.used++
	if p == PriorityBackground {
		a.bgUsed++
	}
	return nil
}

func (a *Admission) Release(p Priority) {
	a.mu.Lock()
	a.used--
	if p == PriorityBackground {
		a.bgUsed--
	}
	a.cond.Broadcast()
	a.mu.Unlock()
}

func (a *Admission) fits(p Priority) bool {
	if a.used >= a.total {
		return false
	}
	if p == PriorityBackground && a.bgUsed >= a.total-a.reserve {
		return false
	}
	return true
}

type circuitState int

const (
	circuitClosed circuitState = iota
	circuitOpen
	circuitHalfOpen
)

// breaker fails fast after repeated provider failures. A half-open probe is
// a single call; cooldown is measured with the injected clock.
type breaker struct {
	mu        sync.Mutex
	state     circuitState
	failures  int
	threshold int
	cooldown  time.Duration
	opened    time.Time
	probe     bool
}

func newBreaker(threshold int, cooldown time.Duration) *breaker {
	return &breaker{threshold: threshold, cooldown: cooldown}
}

func (b *breaker) Allow(now time.Time) error {
	b.mu.Lock()
	defer b.mu.Unlock()
	switch b.state {
	case circuitOpen:
		if now.Sub(b.opened) < b.cooldown {
			return ErrCircuitOpen
		}
		b.state = circuitHalfOpen
		b.probe = true
		return nil
	case circuitHalfOpen:
		if b.probe {
			return ErrCircuitOpen
		}
		b.probe = true
		return nil
	default:
		return nil
	}
}

func (b *breaker) Success() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.state = circuitClosed
	b.failures = 0
	b.probe = false
}

func (b *breaker) Fail(now time.Time) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.probe = false
	if b.state == circuitHalfOpen {
		b.state = circuitOpen
		b.opened = now
		b.failures = b.threshold
		return
	}
	b.failures++
	if b.failures >= b.threshold {
		b.state = circuitOpen
		b.opened = now
	}
}

// Abandon releases a half-open probe that never reached the provider, so a
// later call can try. It does not change a closed or open circuit.
func (b *breaker) Abandon() {
	b.mu.Lock()
	defer b.mu.Unlock()
	if b.state == circuitHalfOpen {
		b.probe = false
	}
}
