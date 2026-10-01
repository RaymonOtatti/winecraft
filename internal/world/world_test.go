package world

import "testing"

func TestRegistryIsConsistent(t *testing.T) {
	seen := map[string]bool{}
	for i, d := range defs {
		if d.ID != TileID(i) {
			t.Fatalf("defs[%d] has ID %d", i, d.ID)
		}
		if d.Name == "" {
			t.Fatalf("tile %d has no name", i)
		}
		if seen[d.Name] {
			t.Fatalf("duplicate tile name %q", d.Name)
		}
		seen[d.Name] = true
		if d.Layer >= NumLayers {
			t.Fatalf("tile %q has invalid layer %d", d.Name, d.Layer)
		}
		if d.Drop != None && int(d.Drop) >= len(defs) {
			t.Fatalf("tile %q drops unknown tile %d", d.Name, d.Drop)
		}
	}
	if Def(None).Walkable {
		t.Fatal("None must not be walkable")
	}
	if Def(TileID(len(defs)+5)).Name != "" {
		t.Fatal("unknown ids must return the zero def")
	}
}

func TestVinesAreHarvestedNotBroken(t *testing.T) {
	if Def(Vine).Breakable {
		t.Fatal("vines must not be breakable: harvesting must never destroy a vineyard")
	}
	if Def(Vine).Layer != Object || Def(VineHarvested).Layer != Object {
		t.Fatal("vines live on the object layer")
	}
}

func TestChunkOf(t *testing.T) {
	cases := []struct {
		x, y           int
		cx, cy, lx, ly int
	}{
		{0, 0, 0, 0, 0, 0},
		{31, 31, 0, 0, 31, 31},
		{32, 0, 1, 0, 0, 0},
		{-1, -1, -1, -1, 31, 31},
		{-32, -33, -1, -2, 0, 31},
		{100, -100, 3, -4, 4, 28},
	}
	for _, c := range cases {
		cc, lx, ly := ChunkOf(c.x, c.y)
		if cc != (ChunkCoord{int32(c.cx), int32(c.cy)}) || lx != c.lx || ly != c.ly {
			t.Errorf("ChunkOf(%d,%d) = %v,%d,%d; want {%d %d},%d,%d", c.x, c.y, cc, lx, ly, c.cx, c.cy, c.lx, c.ly)
		}
	}
}

func TestSetGetAcrossChunkBorders(t *testing.T) {
	w := New()
	points := [][2]int{{31, 0}, {32, 0}, {-1, 0}, {0, -1}, {-33, -65}, {1000, 1000}}
	for _, p := range points {
		w.Set(p[0], p[1], Dirt)
	}
	for _, p := range points {
		if got := w.At(Ground, p[0], p[1]); got != Dirt {
			t.Errorf("At(%v) = %d, want Dirt", p, got)
		}
		if got := w.At(Object, p[0], p[1]); got != None {
			t.Errorf("object layer at %v = %d, want None", p, got)
		}
	}
	// neighbours of written tiles stay empty
	for _, p := range [][2]int{{30, 0}, {33, 0}, {-2, 0}, {0, -2}, {-34, -65}} {
		if got := w.At(Ground, p[0], p[1]); got != None {
			t.Errorf("At(%v) = %d, want None", p, got)
		}
	}
	// (31,0),(32,0),(-1,0),(0,-1),(-33,-65),(1000,1000) span 6 distinct chunks
	if n := w.ChunkCount(); n != 6 {
		t.Errorf("ChunkCount = %d, want 6", n)
	}
}

func TestSetUsesTheTilesOwnLayer(t *testing.T) {
	w := New()
	w.Set(5, 5, Grass)
	w.Set(5, 5, Vine)
	if w.At(Ground, 5, 5) != Grass || w.At(Object, 5, 5) != Vine {
		t.Fatalf("ground=%d object=%d, want Grass and Vine", w.At(Ground, 5, 5), w.At(Object, 5, 5))
	}
	w.Clear(Object, 5, 5)
	if w.At(Object, 5, 5) != None || w.At(Ground, 5, 5) != Grass {
		t.Fatal("Clear(Object) must leave the ground untouched")
	}
}

func TestReadingMissingChunksDoesNotAllocate(t *testing.T) {
	w := New()
	if w.At(Ground, -500, 700) != None {
		t.Fatal("missing chunk must read as None")
	}
	if w.ChunkCount() != 0 {
		t.Fatal("reading must not create chunks")
	}
}
