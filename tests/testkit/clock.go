package testkit

import (
	"sync"
	"time"
)

// FixedClock provides a controllable clock for tests.
type FixedClock struct {
	mu  sync.Mutex
	now time.Time
}

func NewFixedClock(t time.Time) *FixedClock {
	return &FixedClock{now: t}
}

func (c *FixedClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

func (c *FixedClock) Advance(d time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = c.now.Add(d)
}

func (c *FixedClock) Set(t time.Time) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.now = t
}
