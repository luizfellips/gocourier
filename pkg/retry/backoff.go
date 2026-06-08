package retry

import (
	"math/rand"
	"time"
)

func Delay(attempt int, base, max time.Duration) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	d := base * time.Duration(1<<(attempt-1))
	if d > max {
		d = max
	}
	jitter := time.Duration(rand.Int63n(int64(d / 2)))
	return d + jitter
}
