package dispatch

import "time"

// Config holds retry, stream, and circuit-breaker tuning for the dispatch service.
type Config struct {
	MaxAttempts  int
	RetryBase    time.Duration
	RetryMax     time.Duration
	StreamPrefix string
	CBThreshold  int
	CBWindow     time.Duration
	CBCooldown   time.Duration
}
