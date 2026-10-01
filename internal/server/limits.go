package server

import (
	"sync"
	"time"

	"github.com/RaymonOtatti/winecraft/internal/rate"
)

// ipGate tracks, per client IP, how many connections are open and how many
// wrong join codes it sent recently. Connection goroutines share it, so it
// has its own lock (the hub's state is never touched here).
type ipGate struct {
	mu    sync.Mutex
	conns map[string]int
	fails map[string]*rate.Bucket
}

func newIPGate() *ipGate {
	return &ipGate{conns: make(map[string]int), fails: make(map[string]*rate.Bucket)}
}

// enter admits a new connection from ip unless it already has max open.
func (g *ipGate) enter(ip string, max int) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.conns[ip] >= max {
		return false
	}
	g.conns[ip]++
	return true
}

func (g *ipGate) leave(ip string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.conns[ip] <= 1 {
		delete(g.conns, ip)
		return
	}
	g.conns[ip]--
}

// locked reports whether ip has used up its wrong-join-code allowance.
func (g *ipGate) locked(ip string, now time.Time, interval time.Duration, burst int) bool {
	g.mu.Lock()
	defer g.mu.Unlock()
	b := g.fails[ip]
	return b != nil && b.Available(now, interval, burst) < 1
}

// fail records one wrong join code from ip.
func (g *ipGate) fail(ip string, now time.Time, interval time.Duration, burst int) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if len(g.fails) > 4096 {
		// keep memory bounded: forget IPs whose allowance has fully refilled
		for k, b := range g.fails {
			if b.Available(now, interval, burst) >= float64(burst) {
				delete(g.fails, k)
			}
		}
	}
	b := g.fails[ip]
	if b == nil {
		b = new(rate.Bucket)
		g.fails[ip] = b
	}
	b.Take(now, interval, burst, 1)
}
