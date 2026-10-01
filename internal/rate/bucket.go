// Package rate has the token bucket used for every rate limit in WineCraft:
// player steps and edits, messages per connection, and failed joins per IP.
package rate

import "time"

// Bucket holds up to burst tokens and refills one token per interval.
// Callers pass the time, so tests control the clock. The zero value is a
// full bucket on first use.
type Bucket struct {
	tokens float64
	last   time.Time
}

func (b *Bucket) refill(now time.Time, interval time.Duration, burst int) {
	if b.last.IsZero() {
		b.tokens = float64(burst)
		b.last = now
		return
	}
	if el := now.Sub(b.last); el > 0 {
		b.tokens = min(float64(burst), b.tokens+float64(el)/float64(interval))
		b.last = now
	}
}

// Take spends n tokens if they are available and reports whether it did.
func (b *Bucket) Take(now time.Time, interval time.Duration, burst int, n float64) bool {
	b.refill(now, interval, burst)
	if b.tokens < n {
		return false
	}
	b.tokens -= n
	return true
}

// Available refills and returns the tokens on hand without spending any.
func (b *Bucket) Available(now time.Time, interval time.Duration, burst int) float64 {
	b.refill(now, interval, burst)
	return b.tokens
}
