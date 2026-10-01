package game

import (
	"time"

	"github.com/RaymonOtatti/winecraft/internal/world"
)

// Movement speed limits, enforced with a token bucket per player: a burst of
// StepBurst steps absorbs network jitter, and the sustained rate is one step
// per StepInterval. A ledge hop covers two tiles and costs two steps.
const (
	StepInterval = 150 * time.Millisecond
	StepBurst    = 3
)

// Move applies one step requested by a player's client. It always records seq
// and the new facing (a bump turns the player in place, as in Pokémon) and
// marks the player for broadcast, so a client whose prediction was wrong snaps
// back. It reports whether the player actually moved. Moves with a seq that is
// not newer than the last one are replays and are ignored entirely.
func (s *State) Move(id uint32, d world.Dir, seq uint32, now time.Time) bool {
	p := s.players[id]
	if p == nil || !d.Valid() || seq <= p.Seq {
		return false
	}
	p.Seq = seq
	p.Facing = d
	s.dirty[id] = true

	p.refill(now)
	step, ok := s.Map.World.CanStep(p.Pos.X, p.Pos.Y, d)
	if !ok || !s.Map.Bounds.Contains(step.X, step.Y) {
		return false
	}
	cost := 1.0
	if step.Hop {
		cost = 2
	}
	if p.tokens < cost {
		return false
	}
	p.tokens -= cost
	p.Pos = world.Point{X: step.X, Y: step.Y}
	return true
}

func (p *Player) refill(now time.Time) {
	if p.lastRefill.IsZero() {
		p.tokens = StepBurst
		p.lastRefill = now
		return
	}
	if el := now.Sub(p.lastRefill); el > 0 {
		p.tokens = min(StepBurst, p.tokens+float64(el)/float64(StepInterval))
		p.lastRefill = now
	}
}
