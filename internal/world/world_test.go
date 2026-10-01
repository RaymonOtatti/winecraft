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
		if int(d.Drop) >= NumItems() {
			t.Fatalf("tile %q drops unknown item %d", d.Name, d.Drop)
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

func TestPutChunkCopiesTheLayers(t *testing.T) {
	w := New()
	var layers [NumLayers][ChunkSize * ChunkSize]TileID
	layers[Ground][2*ChunkSize+1] = Sand
	w.PutChunk(ChunkCoord{X: -1, Y: 0}, &layers)
	layers[Ground][2*ChunkSize+1] = Water // later changes to the source must not leak in
	if got := w.At(Ground, -32+1, 2); got != Sand {
		t.Fatalf("At = %d, want Sand", got)
	}
}

func TestModifiedTracksChangesSinceTheBaseline(t *testing.T) {
	w := New()
	w.Set(1, 1, Grass)
	w.Set(40, 1, Grass)
	w.MarkBaseline()
	if len(w.Modified()) != 0 {
		t.Fatal("nothing changed since the baseline")
	}
	w.Set(41, 2, Fence)
	w.Clear(Object, -1, -1) // clearing an empty spot in a missing chunk changes nothing
	if got := w.Modified(); len(got) != 1 || got[0] != (ChunkCoord{X: 1, Y: 0}) {
		t.Fatalf("Modified = %v, want just chunk (1,0)", got)
	}
}

func TestGatherableTilesAreConsistent(t *testing.T) {
	for id := 1; id < NumTiles(); id++ {
		d := Def(TileID(id))
		if d.Gather == ItemNone {
			continue
		}
		if d.GatherN < 1 || int(d.Gather) >= NumItems() {
			t.Errorf("%s gathers %d of item %d", d.Name, d.GatherN, d.Gather)
		}
		if d.Spent == None || Def(d.Spent).RegrowsTo != TileID(id) {
			t.Errorf("%s must turn into a spent tile that regrows back into it", d.Name)
		}
		if d.Breakable {
			t.Errorf("%s is gathered, so it must never be breakable (a vineyard is never destroyed)", d.Name)
		}
	}
	if Def(Vine).Gather != ItemGrapes || Def(Vine).Spent != VineHarvested {
		t.Error("vines give grapes and become harvested vines")
	}
}
