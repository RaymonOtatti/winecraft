package client

import (
	"testing"
	"time"

	"github.com/RaymonOtatti/winecraft/internal/client/art"
	"github.com/RaymonOtatti/winecraft/internal/world"
)

// field: grass on [0,10)², a wall at (6,5), a ledge band at y=7; start (5,5).
func field() *world.World {
	w := world.New()
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			w.Set(x, y, world.Grass)
		}
	}
	w.Set(6, 5, world.StoneWall)
	for x := 0; x < 10; x++ {
		w.Set(x, 7, world.LedgeSouth)
	}
	return w
}

const tick = time.Second / 60

// hold feeds direction d for duration dur and collects the moves sent.
func hold(w *Walker, d world.Dir, dur time.Duration) []Move {
	var sent []Move
	for i := 0; i < int(dur/tick); i++ { // whole ticks: summing a rounded tick drifts
		if m, ok := w.Update(tick, d, true); ok {
			sent = append(sent, m)
		}
	}
	return sent
}

func idle(w *Walker, dur time.Duration) {
	for i := 0; i < int(dur/tick); i++ {
		w.Update(tick, 0, false)
	}
}

func newWalker() *Walker {
	w := NewWalker(field(), world.Point{X: 5, Y: 5})
	w.Facing = world.West // already facing west: no turn delay going west
	return w
}

func TestHoldingADirectionWalksTileByTile(t *testing.T) {
	w := newWalker()
	sent := hold(w, world.West, StepDuration/2)
	if len(sent) != 1 || sent[0].Dir != world.West || sent[0].Seq != 1 {
		t.Fatalf("starting a step must send exactly one Move: %+v", sent)
	}
	px, _ := w.DrawPos()
	if want := float64(5*art.Tile) - art.Tile/2; px > want+2 || px < want-2 {
		t.Fatalf("halfway through the step x = %v px, want about %v", px, want)
	}
	hold(w, world.West, StepDuration/2)
	idle(w, StepDuration)
	if w.Pos != (world.Point{X: 4, Y: 5}) || w.Moving() {
		t.Fatalf("after one step: pos %v moving %v, want (4,5) at rest", w.Pos, w.Moving())
	}
}

func TestATapTurnsInPlace(t *testing.T) {
	w := newWalker()
	sent := hold(w, world.North, TurnDelay/2)
	idle(w, StepDuration)
	if w.Facing != world.North || w.Pos != (world.Point{X: 5, Y: 5}) {
		t.Fatalf("a short tap must only turn: facing %v pos %v", w.Facing, w.Pos)
	}
	if len(sent) != 0 {
		t.Fatalf("a turn in place stays local (the server treats every Move as a step): sent %+v", sent)
	}
}

func TestBumpingAWallTurnsAndSendsOnce(t *testing.T) {
	w := newWalker()
	sent := hold(w, world.East, time.Second) // the wall is east
	if w.Pos != (world.Point{X: 5, Y: 5}) || w.Facing != world.East {
		t.Fatalf("bump: pos %v facing %v", w.Pos, w.Facing)
	}
	if len(sent) != 1 {
		t.Fatalf("holding into a wall must send one Move, not %d (the server has a message budget)", len(sent))
	}
}

func TestLedgeHopArcsAndLandsTwoTilesDown(t *testing.T) {
	w := newWalker()
	w.Facing = world.South
	hold(w, world.South, StepDuration) // to (5,6)
	hold(w, world.South, StepDuration) // hop starts
	_, py := w.DrawPos()
	straight := (6 + 2*w.Progress()) * art.Tile // the straight path at this point of the hop
	if py > straight-3 {
		t.Fatalf("mid-hop y = %v, straight path %v: the hop should arc upward", py, straight)
	}
	idle(w, 2*StepDuration)
	if w.Pos != (world.Point{X: 5, Y: 8}) {
		t.Fatalf("after the hop: %v, want (5,8)", w.Pos)
	}
}

func TestWalkFrameAlternatesMidStep(t *testing.T) {
	w := newWalker()
	if w.Frame() != 0 {
		t.Fatal("standing still shows frame 0")
	}
	hold(w, world.West, StepDuration/2)
	if w.Frame() != 1 {
		t.Fatal("mid-step shows the walking frame")
	}
}

func TestReconcileKeepsAConfirmedPrediction(t *testing.T) {
	w := newWalker()
	hold(w, world.West, StepDuration)
	idle(w, StepDuration)
	w.Reconcile(world.Point{X: 4, Y: 5}, world.West, 1)
	if w.Pos != (world.Point{X: 4, Y: 5}) || w.PendingMoves() != 0 {
		t.Fatalf("confirmed move: pos %v pending %d", w.Pos, w.PendingMoves())
	}
}

func TestReconcileSnapsBackAndReplaysPendingMoves(t *testing.T) {
	w := newWalker()
	hold(w, world.West, StepDuration) // seq 1 → predicted (4,5)
	hold(w, world.West, StepDuration) // seq 2 → predicted (3,5)
	idle(w, StepDuration)
	if w.Pos != (world.Point{X: 3, Y: 5}) {
		t.Fatalf("setup: predicted %v", w.Pos)
	}
	// The server refused seq 1 (still at spawn) and has not seen seq 2 yet.
	w.Reconcile(world.Point{X: 5, Y: 5}, world.West, 1)
	if w.Pos != (world.Point{X: 4, Y: 5}) || w.PendingMoves() != 1 {
		t.Fatalf("after the correction: pos %v pending %d, want (4,5) with seq 2 replayed", w.Pos, w.PendingMoves())
	}
}

func TestDpadHitTesting(t *testing.T) {
	d := NewDpad(BaseW, BaseH)
	for want := world.Dir(0); want < 4; want++ {
		r := d.Buttons[want]
		got, ok := d.Hit(r.Min.X+r.Dx()/2, r.Min.Y+r.Dy()/2)
		if !ok || got != want {
			t.Errorf("centre of the %v button gave %v, %v", want, got, ok)
		}
	}
	if _, ok := d.Hit(BaseW/2, BaseH/2); ok {
		t.Error("a touch in the middle of the screen is not a D-pad press")
	}
}

func TestAStepTakesExactlyTwelveTicksAt60Hz(t *testing.T) {
	w := newWalker()
	w.Update(tick, world.West, true) // starts the step
	for i := 0; i < 11; i++ {
		w.Update(tick, world.West, false)
	}
	if !w.Moving() {
		t.Fatal("after 11 ticks the step is still under way")
	}
	w.Update(tick, world.West, false)
	if w.Moving() {
		t.Fatal("tick rounding must not stretch a 200 ms step to 13 ticks")
	}
}
