package circuitbreaker

import (
	"sync"
	"time"

	"github.com/gocourier/pkg/apperrors"
)

type Breaker struct {
	threshold int
	window    time.Duration
	cooldown  time.Duration

	mu        sync.Mutex
	failures  []time.Time
	state     state
	openedAt  time.Time
}

type state int

const (
	stateClosed state = iota
	stateOpen
	stateHalfOpen
)

func New(threshold int, window, cooldown time.Duration) *Breaker {
	return &Breaker{
		threshold: threshold,
		window:    window,
		cooldown:  cooldown,
		state:     stateClosed,
	}
}

func (b *Breaker) Allow() error {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	switch b.state {
	case stateOpen:
		if now.Sub(b.openedAt) >= b.cooldown {
			b.state = stateHalfOpen
			return nil
		}
		return apperrors.ErrCircuitOpen
	case stateHalfOpen, stateClosed:
		return nil
	default:
		return nil
	}
}

func (b *Breaker) RecordSuccess() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures = nil
	b.state = stateClosed
}

func (b *Breaker) RecordFailure() {
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-b.window)
	var recent []time.Time
	for _, t := range b.failures {
		if t.After(cutoff) {
			recent = append(recent, t)
		}
	}
	recent = append(recent, now)
	b.failures = recent

	if b.state == stateHalfOpen || len(b.failures) >= b.threshold {
		b.state = stateOpen
		b.openedAt = now
	}
}
