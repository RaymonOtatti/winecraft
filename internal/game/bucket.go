package game

import "time"

// bucket is a token bucket: it holds up to burst tokens and refills one token
// per interval. Callers pass the time, so tests control the clock.
type bucket struct {
	tokens float64
	last   time.Time
}

// take spends n tokens if available and reports whether it did.
func (b *bucket) take(now time.Time, interval time.Duration, burst int, n float64) bool {
	if b.last.IsZero() {
		b.tokens = float64(burst)
		b.last = now
	} else if el := now.Sub(b.last); el > 0 {
		b.tokens = min(float64(burst), b.tokens+float64(el)/float64(interval))
		b.last = now
	}
	if b.tokens < n {
		return false
	}
	b.tokens -= n
	return true
}
