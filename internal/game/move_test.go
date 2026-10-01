package game

import (
	"testing"
	"time"

	"github.com/RaymonOtatti/winecraft/internal/world"
)

// tinyMap: grass on [0,10)², a wall east of spawn, a ledge band at y=7.
//
//	y=5  . . . . . @ W . . .     @ spawn (5,5), W stone wall (6,5)
//	y=7  v v v v v v v v v v     ledge, hop south only
func tinyMap() *world.DevMap {
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
	return &world.DevMap{World: w, Bounds: world.Rect{W: 10, H: 10}, Spawn: world.Point{X: 5, Y: 5}}
}

var t0 = time.Date(2026, 10, 1, 12, 0, 0, 0, time.UTC)

func joined(t *testing.T) (*State, *Player) {
	t.Helper()
	s := New(tinyMap())
	p, err := s.Join("Franco")
	if err != nil {
		t.Fatal(err)
	}
	s.TakeDirty()
	return s, p
}

func TestMoveAdvancesAndRecordsSeq(t *testing.T) {
	s, p := joined(t)
	if !s.Move(p.ID, world.West, 1, t0) {
		t.Fatal("a step onto grass must be accepted")
	}
	if p.Pos != (world.Point{X: 4, Y: 5}) || p.Facing != world.West || p.Seq != 1 {
		t.Fatalf("after move: pos %v facing %v seq %d", p.Pos, p.Facing, p.Seq)
	}
	if d := s.TakeDirty(); len(d) != 1 {
		t.Fatal("a move must mark the player dirty for broadcast")
	}
}

func TestWallBumpTurnsButDoesNotMove(t *testing.T) {
	s, p := joined(t)
	p.Facing = world.North
	if s.Move(p.ID, world.East, 1, t0) {
		t.Fatal("walking into a wall must be rejected")
	}
	if p.Pos != (world.Point{X: 5, Y: 5}) {
		t.Fatalf("player moved into the wall: %v", p.Pos)
	}
	if p.Facing != world.East || p.Seq != 1 {
		t.Fatal("a bump still turns the player and records the seq, so the client can reconcile")
	}
	if len(s.TakeDirty()) != 1 {
		t.Fatal("a rejected move must be broadcast so the client snaps back")
	}
}

func TestLedgeHopsDownButNeverUp(t *testing.T) {
	s, p := joined(t)
	if !s.Move(p.ID, world.South, 1, t0) {
		t.Fatal("step to (5,6)")
	}
	if !s.Move(p.ID, world.South, 2, t0.Add(time.Second)) || p.Pos != (world.Point{X: 5, Y: 8}) {
		t.Fatalf("hop over the ledge should land at (5,8), got %v", p.Pos)
	}
	if s.Move(p.ID, world.North, 3, t0.Add(2*time.Second)) || p.Pos != (world.Point{X: 5, Y: 8}) {
		t.Fatalf("climbing the ledge must be rejected, player at %v", p.Pos)
	}
}

func TestSpeedHackIsCapped(t *testing.T) {
	s, p := joined(t)
	accepted := 0
	for seq := uint32(1); seq <= 10; seq++ {
		if s.Move(p.ID, world.West, seq, t0) { // ten steps in the same instant
			accepted++
		}
	}
	if accepted != StepBurst {
		t.Fatalf("accepted %d instant steps, want the burst of %d", accepted, StepBurst)
	}
	if !s.Move(p.ID, world.North, 11, t0.Add(StepInterval)) {
		t.Fatal("one more step must be allowed after one step interval")
	}
	if s.Move(p.ID, world.North, 12, t0.Add(StepInterval)) {
		t.Fatal("but not two")
	}
}

func TestSustainedSpeedIsOneStepPerInterval(t *testing.T) {
	s, p := joined(t)
	now := t0
	steps := 0
	// try to move every 50 ms for 3 s, back and forth so the map never runs out
	for i := 0; i < 60; i++ {
		d := world.West
		if (i/4)%2 == 1 {
			d = world.East
		}
		if s.Move(p.ID, d, uint32(i+1), now) {
			steps++
		}
		now = now.Add(50 * time.Millisecond)
	}
	max := StepBurst + int(3*time.Second/StepInterval)
	if steps > max {
		t.Fatalf("%d steps in 3 s, the cap is %d", steps, max)
	}
}

func TestHopCostsTwoSteps(t *testing.T) {
	s, p := joined(t)
	s.Move(p.ID, world.South, 1, t0)                       // 1 token used, 2 left
	if !s.Move(p.ID, world.South, 2, t0) || p.Pos.Y != 8 { // hop: 2 tokens, 0 left
		t.Fatalf("hop with two tokens left should succeed, at %v", p.Pos)
	}
	if s.Move(p.ID, world.West, 3, t0) {
		t.Fatal("after a hop the bucket is empty")
	}
}

func TestStaleOrDuplicateSeqIsIgnored(t *testing.T) {
	s, p := joined(t)
	s.Move(p.ID, world.West, 5, t0)
	pos := p.Pos
	if s.Move(p.ID, world.West, 5, t0.Add(time.Second)) || s.Move(p.ID, world.West, 4, t0.Add(time.Second)) {
		t.Fatal("a replayed or older seq must be ignored")
	}
	if p.Pos != pos || p.Seq != 5 {
		t.Fatal("ignored moves must not change anything")
	}
}

func TestMoveByUnknownPlayerOrBadDirIsIgnored(t *testing.T) {
	s, p := joined(t)
	if s.Move(999, world.West, 1, t0) {
		t.Fatal("unknown player moved")
	}
	if s.Move(p.ID, world.Dir(7), 1, t0) {
		t.Fatal("invalid direction accepted")
	}
}
