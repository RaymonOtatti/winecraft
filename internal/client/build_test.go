package client

import (
	"testing"

	"github.com/RaymonOtatti/winecraft/internal/proto"
	"github.com/RaymonOtatti/winecraft/internal/world"
)

func TestHotbarHoldsThePlaceableTiles(t *testing.T) {
	h := NewHotbar()
	if len(h.Slots) == 0 {
		t.Fatal("empty hotbar")
	}
	for _, id := range h.Slots {
		if !world.Def(id).Placeable {
			t.Errorf("slot holds %s, which cannot be placed", world.Def(id).Name)
		}
	}
	h.Select(1)
	if h.Selected() != h.Slots[1] {
		t.Fatal("Select must pick the slot")
	}
	h.Select(99)
	if h.Selected() != h.Slots[1] {
		t.Fatal("selecting a slot that does not exist must change nothing")
	}
	h.Next()
	h.Prev()
	h.Prev()
	if h.Selected() != h.Slots[0] {
		t.Fatal("Next/Prev must step through the slots")
	}
	h.Prev()
	if h.Selected() != h.Slots[len(h.Slots)-1] {
		t.Fatal("Prev from the first slot wraps to the last")
	}
}

// builder: a welcomed session at (5,5) facing east, sandbox x<8.
func builder(t *testing.T) *Session {
	t.Helper()
	s := welcomed(t)
	s.Sandbox = world.Rect{W: 8, H: 10}
	s.Me.Facing = world.East
	return s
}

func TestPlaceTargetsTheFacedTile(t *testing.T) {
	s := builder(t)
	e, ok := s.EditFor(ActionBuild, world.Fence)
	if !ok || *e != (proto.Edit{X: 6, Y: 5, Layer: world.Object, Tile: world.Fence}) {
		t.Fatalf("place fence: %+v ok=%v", e, ok)
	}
	e, ok = s.EditFor(ActionBuild, world.Planks)
	if !ok || e.Layer != world.Ground {
		t.Fatalf("planks go on the ground layer: %+v", e)
	}
}

func TestBreakPicksTheObjectFirstThenTheFloor(t *testing.T) {
	s := builder(t)
	if _, ok := s.EditFor(ActionBreak, 0); ok {
		t.Fatal("breaking plain grass must not send anything")
	}
	s.World.Set(6, 5, world.Planks)
	s.World.Set(6, 5, world.Crate)
	if e, ok := s.EditFor(ActionBreak, 0); !ok || e.Layer != world.Object || e.Tile != world.None {
		t.Fatalf("break with a crate on planks: %+v ok=%v, want the crate", e, ok)
	}
	s.World.Clear(world.Object, 6, 5)
	if e, ok := s.EditFor(ActionBreak, 0); !ok || e.Layer != world.Ground {
		t.Fatalf("break on bare planks: %+v ok=%v, want the floor", e, ok)
	}
}

func TestNoEditsOutsideTheSandboxOrMidStep(t *testing.T) {
	s := builder(t)
	s.Me.Reset(world.Point{X: 7, Y: 5}) // facing east → target (8,5), outside
	s.Me.Facing = world.East
	if _, ok := s.EditFor(ActionBuild, world.Fence); ok {
		t.Fatal("must not send edits for tiles outside the sandbox")
	}
	s.Me.Reset(world.Point{X: 5, Y: 5})
	s.Me.Facing = world.West
	hold(s.Me, world.West, StepDuration/2) // mid-step
	if _, ok := s.EditFor(ActionBuild, world.Fence); ok {
		t.Fatal("must not build while walking")
	}
}
