package game

import (
	"errors"
	"testing"

	"github.com/RaymonOtatti/winecraft/internal/world"
)

// editMap: grass on [0,10)², the sandbox is the west part x<6, spawn (5,5)
// sits on its east edge.
//
//	y=4  . . . . . R | . . .     R rock (breakable)
//	y=5  . . . . . @ | V . .     @ spawn; V a vine at (6,5), outside the sandbox
//	y=6  . . . . . P | . . .     P poplar (not breakable)
func editMap() *world.DevMap {
	w := world.New()
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			w.Set(x, y, world.Grass)
		}
	}
	w.Set(5, 4, world.Rock)
	w.Set(5, 6, world.Poplar)
	w.Set(6, 5, world.Dirt)
	w.Set(6, 5, world.Vine) // outside the sandbox: harvesting works anywhere
	return &world.DevMap{
		World:   w,
		Bounds:  world.Rect{W: 10, H: 10},
		Spawn:   world.Point{X: 5, Y: 5},
		Sandbox: world.Rect{X: 0, Y: 0, W: 6, H: 10},
	}
}

func editor(t *testing.T) (*State, *Player) {
	t.Helper()
	s := New(editMap())
	p, err := s.Join("Franco")
	if err != nil {
		t.Fatal(err)
	}
	return s, p
}

func TestPlaceAndBreakInsideTheSandbox(t *testing.T) {
	s, p := editor(t)
	ch, err := s.Edit(p.ID, 4, 5, world.Object, world.Fence, t0)
	if err != nil {
		t.Fatalf("placing a fence next to you in the sandbox: %v", err)
	}
	if ch != (Change{X: 4, Y: 5, Layer: world.Object, Tile: world.Fence}) || s.Map.World.At(world.Object, 4, 5) != world.Fence {
		t.Fatalf("change %+v, world has %d", ch, s.Map.World.At(world.Object, 4, 5))
	}
	if _, err := s.Edit(p.ID, 4, 5, world.Object, world.None, t0); err != nil {
		t.Fatalf("breaking your own fence: %v", err)
	}
	if s.Map.World.At(world.Object, 4, 5) != world.None {
		t.Fatal("fence still there after breaking it")
	}
	if _, err := s.Edit(p.ID, 5, 4, world.Object, world.None, t0); err != nil {
		t.Fatalf("breaking a rock: %v", err)
	}
}

func TestFloorsReplaceSoilAndBreakBackToDirt(t *testing.T) {
	s, p := editor(t)
	if _, err := s.Edit(p.ID, 4, 5, world.Ground, world.Planks, t0); err != nil {
		t.Fatalf("laying planks on grass: %v", err)
	}
	if s.Map.World.At(world.Ground, 4, 5) != world.Planks {
		t.Fatal("planks not laid")
	}
	ch, err := s.Edit(p.ID, 4, 5, world.Ground, world.None, t0)
	if err != nil {
		t.Fatalf("lifting planks: %v", err)
	}
	if ch.Tile != world.Dirt || s.Map.World.At(world.Ground, 4, 5) != world.Dirt {
		t.Fatal("breaking a floor must leave dirt, never a hole in the world")
	}
}

func TestEditsThatAreNotAllowed(t *testing.T) {
	cases := []struct {
		name  string
		x, y  int
		layer world.Layer
		tile  world.TileID
	}{
		{"outside the sandbox", 6, 5, world.Object, world.Fence},
		{"out of reach", 3, 5, world.Object, world.Fence},
		{"on your own tile", 5, 5, world.Object, world.Fence},
		{"onto an occupied tile", 5, 4, world.Object, world.Fence},
		{"break an unbreakable poplar", 5, 6, world.Object, world.None},
		{"break nothing", 4, 5, world.Object, world.None},
		{"break natural grass", 4, 5, world.Ground, world.None},
		{"place a vine", 4, 5, world.Object, world.Vine},
		{"tile on the wrong layer", 4, 5, world.Ground, world.Fence},
		{"invalid layer", 4, 5, world.NumLayers, world.Fence},
	}
	for _, c := range cases {
		s, p := editor(t)
		before := s.Map.World.Digest()
		if _, err := s.Edit(p.ID, c.x, c.y, c.layer, c.tile, t0); !errors.Is(err, ErrNotAllowed) {
			t.Errorf("%s: err = %v, want ErrNotAllowed", c.name, err)
		}
		if s.Map.World.Digest() != before {
			t.Errorf("%s: a rejected edit changed the world", c.name)
		}
	}
}

func TestCannotBuildOnAnotherPlayer(t *testing.T) {
	s, p := editor(t)
	q, _ := s.Join("Raymon")
	if !s.Move(q.ID, world.West, 1, t0) { // Raymon steps to (4,5)
		t.Fatal("setup move failed")
	}
	if _, err := s.Edit(p.ID, 4, 5, world.Object, world.StoneWall, t0); !errors.Is(err, ErrNotAllowed) {
		t.Fatalf("walling in another player: err = %v, want ErrNotAllowed", err)
	}
}

func TestEditFloodIsRateLimited(t *testing.T) {
	s, p := editor(t)
	tile := []world.TileID{world.Fence, world.None}
	for i := 0; i < EditBurst; i++ {
		if _, err := s.Edit(p.ID, 4, 5, world.Object, tile[i%2], t0); err != nil {
			t.Fatalf("edit %d inside the burst: %v", i, err)
		}
	}
	next := tile[EditBurst%2]
	if _, err := s.Edit(p.ID, 4, 5, world.Object, next, t0); !errors.Is(err, ErrRateLimited) {
		t.Fatalf("edit past the burst: err = %v, want ErrRateLimited", err)
	}
	if _, err := s.Edit(p.ID, 4, 5, world.Object, next, t0.Add(EditInterval)); err != nil {
		t.Fatalf("one edit per interval must be allowed: %v", err)
	}
}

func TestEditByUnknownPlayer(t *testing.T) {
	s, _ := editor(t)
	if _, err := s.Edit(999, 4, 5, world.Object, world.Fence, t0); !errors.Is(err, ErrNotAllowed) {
		t.Fatalf("err = %v, want ErrNotAllowed", err)
	}
}
