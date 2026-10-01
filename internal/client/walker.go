package client

import (
	"math"
	"time"

	"github.com/RaymonOtatti/winecraft/internal/client/art"
	"github.com/RaymonOtatti/winecraft/internal/world"
)

// Walking feel. StepDuration must stay above the server's StepInterval
// (150 ms) so honest clients never hit the speed cap.
const (
	StepDuration = 200 * time.Millisecond
	TurnDelay    = 90 * time.Millisecond // a tap shorter than this only turns
	hopArc       = 8                     // pixels a ledge hop rises

	// slack absorbs the rounding of a 60 Hz tick (16,666,666 ns): twelve
	// ticks fall 8 ns short of 200 ms and would otherwise cost a thirteenth.
	slack = 100 * time.Microsecond
)

// Move is a step attempt to send to the server.
type Move struct {
	Dir world.Dir
	Seq uint32
}

// Walker moves the local player Pokémon-style: one tile per step with smooth
// interpolation. It predicts with the same world.CanStep the server enforces
// and keeps the moves the server has not confirmed yet, so Reconcile can
// correct a wrong prediction.
type Walker struct {
	World  *world.World
	Pos    world.Point // the tile we stand on, or are stepping onto
	Facing world.Dir

	from     world.Point
	moving   bool
	elapsed  time.Duration
	dur      time.Duration
	turning  bool // turned from standing; waiting TurnDelay before stepping
	held     time.Duration
	bumpSent bool // the server already knows about this bump
	seq      uint32
	pending  []Move
}

// NewWalker starts at pos facing south.
func NewWalker(w *world.World, pos world.Point) *Walker {
	return &Walker{World: w, Pos: pos, Facing: world.South}
}

// Update advances by dt with direction dir held (or nothing held) and
// returns a Move when a step attempt must go to the server. Turning in place
// stays local: the server treats every Move as a step.
func (w *Walker) Update(dt time.Duration, dir world.Dir, held bool) (Move, bool) {
	arrived := false
	if w.moving {
		w.elapsed += dt
		if w.elapsed+slack < w.dur {
			return Move{}, false
		}
		w.moving, w.elapsed = false, 0
		arrived = true
	}
	if !held || !dir.Valid() {
		w.turning, w.held, w.bumpSent = false, 0, false
		return Move{}, false
	}
	if dir != w.Facing {
		w.Facing = dir
		w.bumpSent = false
		if !arrived { // from standing: turn first, step only if still held
			w.turning, w.held = true, 0
			return Move{}, false
		}
	}
	if w.turning {
		w.held += dt
		if w.held < TurnDelay {
			return Move{}, false
		}
		w.turning = false
	}

	step, ok := w.World.CanStep(w.Pos.X, w.Pos.Y, dir)
	if !ok {
		if w.bumpSent {
			return Move{}, false
		}
		w.bumpSent = true
		return w.send(dir), true // the server records the bump and our facing
	}
	w.bumpSent = false
	w.from, w.Pos = w.Pos, world.Point{X: step.X, Y: step.Y}
	w.moving, w.elapsed, w.dur = true, 0, StepDuration
	if step.Hop {
		w.dur = 2 * StepDuration
	}
	return w.send(dir), true
}

func (w *Walker) send(d world.Dir) Move {
	w.seq++
	m := Move{Dir: d, Seq: w.seq}
	w.pending = append(w.pending, m)
	return m
}

// Reconcile applies the server's authoritative state for us: position after
// it processed our move ack. Moves the server has not seen yet are replayed
// on top; if that lands somewhere other than our prediction, we snap there.
func (w *Walker) Reconcile(server world.Point, facing world.Dir, ack uint32) {
	i := 0
	for i < len(w.pending) && w.pending[i].Seq <= ack {
		i++
	}
	w.pending = w.pending[i:]
	pos := server
	for _, m := range w.pending {
		if s, ok := w.World.CanStep(pos.X, pos.Y, m.Dir); ok {
			pos = world.Point{X: s.X, Y: s.Y}
		}
	}
	if pos != w.Pos {
		w.Pos, w.moving, w.elapsed = pos, false, 0
		if len(w.pending) == 0 {
			w.Facing = facing
		}
	}
}

// Reset puts the walker at pos, at rest, forgetting unconfirmed moves (a new
// Welcome after a reconnect: the server's state starts over).
func (w *Walker) Reset(pos world.Point) {
	w.Pos, w.from = pos, pos
	w.moving, w.elapsed, w.turning, w.held, w.bumpSent = false, 0, false, 0, false
	w.pending = w.pending[:0]
}

// PendingMoves is how many sent moves the server has not confirmed.
func (w *Walker) PendingMoves() int { return len(w.pending) }

// Moving reports whether a step is in progress.
func (w *Walker) Moving() bool { return w.moving }

// Progress is how far through the current step we are, 0 to 1.
func (w *Walker) Progress() float64 {
	if !w.moving {
		return 0
	}
	return float64(w.elapsed) / float64(w.dur)
}

// DrawPos is the interpolated world-pixel position of the sprite's top-left.
func (w *Walker) DrawPos() (float64, float64) {
	if !w.moving {
		return float64(w.Pos.X * art.Tile), float64(w.Pos.Y * art.Tile)
	}
	p := w.Progress()
	x := (float64(w.from.X) + float64(w.Pos.X-w.from.X)*p) * art.Tile
	y := (float64(w.from.Y) + float64(w.Pos.Y-w.from.Y)*p) * art.Tile
	if w.dur > StepDuration { // a ledge hop
		y -= math.Sin(math.Pi*p) * hopArc
	}
	return x, y
}

// Frame is the walk-animation frame: 1 in the middle of a step, else 0.
func (w *Walker) Frame() int {
	if p := w.Progress(); w.moving && p >= 0.25 && p < 0.75 {
		return 1
	}
	return 0
}
