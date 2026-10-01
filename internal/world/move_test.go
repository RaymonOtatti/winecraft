package world

import "testing"

// grassField returns a world whose ground is grass on [0,10)×[0,10).
func grassField() *World {
	w := New()
	for y := 0; y < 10; y++ {
		for x := 0; x < 10; x++ {
			w.Set(x, y, Grass)
		}
	}
	return w
}

func TestCanStepBasicTerrain(t *testing.T) {
	w := grassField()
	w.Set(6, 5, Water)
	w.Set(5, 4, Grass)
	w.Set(5, 4, StoneWall)
	w.Set(4, 5, Bridge)
	w.Set(5, 6, Vine)

	cases := []struct {
		name string
		x, y int
		d    Dir
		ok   bool
	}{
		{"onto bridge", 5, 5, West, true},
		{"onto water", 5, 5, East, false},
		{"into a wall", 5, 5, North, false},
		{"into a vine row", 5, 5, South, false},
		{"off the map edge into the void", 0, 0, North, false},
		{"plain grass", 2, 2, East, true},
	}
	for _, c := range cases {
		s, ok := w.CanStep(c.x, c.y, c.d)
		if ok != c.ok {
			t.Errorf("%s: ok = %v, want %v", c.name, ok, c.ok)
			continue
		}
		if ok {
			dx, dy := c.d.Delta()
			if s.X != c.x+dx || s.Y != c.y+dy || s.Hop {
				t.Errorf("%s: step = %+v, want one tile to (%d,%d)", c.name, s, c.x+dx, c.y+dy)
			}
		}
	}
}

func TestLedgesAreOneWay(t *testing.T) {
	w := grassField()
	for x := 0; x < 10; x++ {
		w.Set(x, 5, LedgeSouth) // a ledge band along y=5
	}

	// heading south from above the band: hop over it and land two tiles down
	s, ok := w.CanStep(3, 4, South)
	if !ok || !s.Hop || s.X != 3 || s.Y != 6 {
		t.Fatalf("south over ledge = %+v ok=%v, want hop to (3,6)", s, ok)
	}

	// climbing back up is not allowed
	if _, ok := w.CanStep(3, 6, North); ok {
		t.Fatal("must not climb a ledge heading north")
	}
	// walking along the ledge band is not allowed
	if _, ok := w.CanStep(2, 5, East); ok {
		t.Fatal("must not walk along a ledge")
	}
	if _, ok := w.CanStep(4, 4, South); !ok {
		t.Fatal("any column of the band can be hopped")
	}
}

func TestLedgeHopNeedsAFreeLandingTile(t *testing.T) {
	w := grassField()
	w.Set(3, 5, LedgeSouth)
	w.Set(3, 6, Rock)
	if _, ok := w.CanStep(3, 4, South); ok {
		t.Fatal("hop must fail when a rock blocks the landing tile")
	}
	w.Clear(Object, 3, 6)
	w.Set(3, 6, Water)
	if _, ok := w.CanStep(3, 4, South); ok {
		t.Fatal("hop must fail when the landing tile is water")
	}
	w.Set(3, 6, LedgeSouth)
	if _, ok := w.CanStep(3, 4, South); ok {
		t.Fatal("hop must not land on another ledge")
	}
}

func TestDirDelta(t *testing.T) {
	want := map[Dir][2]int{North: {0, -1}, South: {0, 1}, West: {-1, 0}, East: {1, 0}}
	for d, v := range want {
		dx, dy := d.Delta()
		if dx != v[0] || dy != v[1] {
			t.Errorf("%v.Delta() = %d,%d want %v", d, dx, dy, v)
		}
		if !d.Valid() {
			t.Errorf("%v must be valid", d)
		}
	}
	if Dir(9).Valid() {
		t.Error("Dir(9) must be invalid")
	}
	if _, ok := grassField().CanStep(3, 3, Dir(9)); ok {
		t.Error("an invalid direction must never step")
	}
}
